package database

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"

	"github.com/armadaproject/armada/internal/common/armadacontext"
	"github.com/armadaproject/armada/internal/common/compress"
	"github.com/armadaproject/armada/internal/common/database"
	protoutil "github.com/armadaproject/armada/internal/common/proto"
	armadaslices "github.com/armadaproject/armada/internal/common/slices"
	"github.com/armadaproject/armada/pkg/armadaevents"
)

// hasSerial is an Interface for db objects that have serial numbers
type hasSerial interface {
	GetSerial() int64
}

type JobRunLease struct {
	RunID                  string
	Queue                  string
	Pool                   string
	JobSet                 string
	UserID                 string
	Node                   string
	Groups                 []byte
	SubmitMessage          []byte
	PodRequirementsOverlay []byte
}

// JobRepository is an interface to be implemented by structs which provide job and run information
type JobRepository interface {
	// FetchInitialJobs returns all non-terminal jobs and their associated job runs as well as the maximum job and run
	// serials in the event that there are no non-terminal jobs/runs.
	FetchInitialJobs(ctx *armadacontext.Context) ([]Job, []Run, *int64, *int64, error)

	// FetchJobUpdates returns all jobs and job dbRuns that have been updated after jobSerial and jobRunSerial respectively
	// These updates are guaranteed to be consistent with each other
	FetchJobUpdates(ctx *armadacontext.Context, jobSerial int64, jobRunSerial int64) ([]Job, []Run, error)

	// FetchJobRunErrors returns all armadaevents.JobRunErrors for the provided job run ids. The returned map is
	// keyed by job run id. Any dbRuns which don't have errors wil be absent from the map.
	FetchJobRunErrors(ctx *armadacontext.Context, runIds []string) (map[string]*armadaevents.Error, error)

	// CountReceivedPartitions returns a count of the number of partition messages present in the database corresponding
	// to the provided groupId.  This is used by the scheduler to determine if the database represents the state of
	// pulsar after a given point in time.
	CountReceivedPartitions(ctx *armadacontext.Context, groupId uuid.UUID) (uint32, error)

	// FindInactiveRuns returns a slice containing all dbRuns that the scheduler does not currently consider active
	// Runs are inactive if they don't exist or if they have succeeded, failed or been cancelled
	FindInactiveRuns(ctx *armadacontext.Context, runIds []string) ([]string, error)

	// FetchJobRunLeases fetches new job runs for a given executor.  A maximum of maxResults rows will be returned, while run
	// in excludedRunIds will be excluded
	FetchJobRunLeases(ctx *armadacontext.Context, executor string, maxResults uint, excludedRunIds []string) ([]*JobRunLease, error)
}

// PostgresJobRepository is an implementation of JobRepository that stores its state in postgres
type PostgresJobRepository struct {
	// pool of database connections
	db *pgxpool.Pool
	// maximum number of rows to fetch from postgres in a single query
	batchSize int32
}

func NewPostgresJobRepository(db *pgxpool.Pool, batchSize int32) *PostgresJobRepository {
	return &PostgresJobRepository{
		db:        db,
		batchSize: batchSize,
	}
}

func (r *PostgresJobRepository) FetchInitialJobs(ctx *armadacontext.Context) ([]Job, []Run, *int64, *int64, error) {
	var initialJobs []Job
	var initialRuns []Run
	var maxJobSerial *int64
	var maxRunSerial *int64

	start := time.Now()
	defer func() {
		ctx.Infof(
			"received %d initial jobs and %d initial job runs from postgres in %s",
			len(initialJobs), len(initialRuns), time.Since(start),
		)
	}()

	// Use a RepeatableRead transaction here so that we get consistency between jobs and dbRuns
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.RepeatableRead,
		AccessMode:     pgx.ReadOnly,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		var err error
		queries := New(tx)

		// Fetch jobs
		loggerCtx := armadacontext.New(ctx, ctx.Logger().WithField("query", "initial-jobs"))
		initialJobRows, err := fetch(loggerCtx, 0, r.batchSize, func(from int64) ([]SelectInitialJobsRow, error) {
			return queries.SelectInitialJobs(ctx, SelectInitialJobsParams{Serial: from, Limit: r.batchSize})
		})
		if err != nil {
			return fmt.Errorf("selecting initial jobs: %w", err)
		}

		initialJobs = make([]Job, len(initialJobRows))
		updatedJobIds := make([]string, len(initialJobRows))
		for i, row := range initialJobRows {
			updatedJobIds[i] = row.JobID
			initialJobs[i] = Job{
				JobID:                   row.JobID,
				JobSet:                  row.JobSet,
				Queue:                   row.Queue,
				Priority:                row.Priority,
				Submitted:               row.Submitted,
				Validated:               row.Validated,
				Queued:                  row.Queued,
				QueuedVersion:           row.QueuedVersion,
				CancelRequested:         row.CancelRequested,
				CancelUser:              row.CancelUser,
				Cancelled:               row.Cancelled,
				CancelByJobsetRequested: row.CancelByJobsetRequested,
				Succeeded:               row.Succeeded,
				Failed:                  row.Failed,
				SchedulingInfo:          row.SchedulingInfo,
				SchedulingInfoVersion:   row.SchedulingInfoVersion,
				Serial:                  row.Serial,
				Pools:                   row.Pools,
				PriceBand:               row.PriceBand,
			}
		}

		// Fetch dbRuns
		loggerCtx = armadacontext.New(ctx, ctx.Logger().WithField("query", "initial-runs"))
		initialRuns, err = fetch(loggerCtx, 0, r.batchSize, func(from int64) ([]Run, error) {
			return queries.SelectInitialRuns(ctx, SelectInitialRunsParams{Serial: from, Limit: r.batchSize, JobIds: updatedJobIds})
		})
		if err != nil {
			return fmt.Errorf("selecting initial runs: %w", err)
		}

		// Hit a case where the database is empty or all rows are in a terminal state. In the event that all rows are
		// in a terminal state return the max job/run serial.
		if len(initialJobs) == 0 {
			maybeMaxJobSerial, err := queries.SelectMaxJobSerial(ctx)
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) { // Ignore errors when the DB is empty
					return fmt.Errorf("selecting max job serial: %w", err)
				}
			} else {
				maxJobSerial = &maybeMaxJobSerial
			}
		}

		if len(initialRuns) == 0 {
			maybeMaxRunSerial, err := queries.SelectMaxRunSerial(ctx)
			if err != nil {
				if !errors.Is(err, pgx.ErrNoRows) { // Ignore errors when the DB is empty
					return fmt.Errorf("selecting max run serial: %w", err)
				}
			} else {
				maxRunSerial = &maybeMaxRunSerial
			}
		}

		return nil
	})

	return initialJobs, initialRuns, maxJobSerial, maxRunSerial, err
}

// FetchJobRunErrors returns all armadaevents.JobRunErrors for the provided job run ids.  The returned map is
// keyed by job run id.  Any dbRuns which don't have errors wil be absent from the map.
func (r *PostgresJobRepository) FetchJobRunErrors(ctx *armadacontext.Context, runIds []string) (map[string]*armadaevents.Error, error) {
	if len(runIds) == 0 {
		return map[string]*armadaevents.Error{}, nil
	}

	chunks := armadaslices.PartitionToMaxLen(runIds, int(r.batchSize))

	errorsByRunId := make(map[string]*armadaevents.Error, len(runIds))
	decompressor := compress.NewZlibDecompressor()

	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		for _, chunk := range chunks {
			tmpTable, err := insertRunIdsToTmpTable(ctx, tx, chunk)
			if err != nil {
				return err
			}

			query := `
		SELECT  job_run_errors.run_id, job_run_errors.error
		FROM %s as tmp
		JOIN job_run_errors ON job_run_errors.run_id = tmp.run_id`

			rows, err := tx.Query(ctx, fmt.Sprintf(query, tmpTable))
			if err != nil {
				return err
			}
			defer rows.Close()
			for rows.Next() {
				var runId string
				var errorBytes []byte
				err := rows.Scan(&runId, &errorBytes)
				if err != nil {
					return errors.WithStack(err)
				}
				jobError, err := protoutil.DecompressAndUnmarshall(errorBytes, &armadaevents.Error{}, decompressor)
				if err != nil {
					return errors.WithStack(err)
				}
				errorsByRunId[runId] = jobError
			}
		}
		return nil
	})

	return errorsByRunId, err
}

// FetchJobUpdates returns all jobs and job dbRuns that have been updated after jobSerial and jobRunSerial respectively
// These updates are guaranteed to be consistent with each other
func (r *PostgresJobRepository) FetchJobUpdates(ctx *armadacontext.Context, jobSerial int64, jobRunSerial int64) ([]Job, []Run, error) {
	var updatedJobs []Job = nil
	var updatedRuns []Run = nil

	start := time.Now()
	defer func() {
		ctx.Infof(
			"received %d updated jobs and %d updated job runs from postgres in %s",
			len(updatedJobs), len(updatedRuns), time.Since(start),
		)
	}()

	// Use a RepeatableRead transaction here so that we get consistency between jobs and dbRuns
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.RepeatableRead,
		AccessMode:     pgx.ReadOnly,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		var err error
		queries := New(tx)

		// Fetch jobs
		loggerCtx := armadacontext.New(ctx, ctx.Logger().WithField("query", "updated-jobs"))
		updatedJobRows, err := fetch(loggerCtx, jobSerial, r.batchSize, func(from int64) ([]SelectUpdatedJobsRow, error) {
			return queries.SelectUpdatedJobs(ctx, SelectUpdatedJobsParams{Serial: from, Limit: r.batchSize})
		})
		updatedJobs = make([]Job, len(updatedJobRows))

		for i, row := range updatedJobRows {
			updatedJobs[i] = Job{
				JobID:                   row.JobID,
				JobSet:                  row.JobSet,
				Queue:                   row.Queue,
				Priority:                row.Priority,
				Submitted:               row.Submitted,
				Validated:               row.Validated,
				Queued:                  row.Queued,
				QueuedVersion:           row.QueuedVersion,
				CancelRequested:         row.CancelRequested,
				Cancelled:               row.Cancelled,
				CancelByJobsetRequested: row.CancelByJobsetRequested,
				CancelUser:              row.CancelUser,
				Succeeded:               row.Succeeded,
				Failed:                  row.Failed,
				SchedulingInfo:          row.SchedulingInfo,
				SchedulingInfoVersion:   row.SchedulingInfoVersion,
				Serial:                  row.Serial,
				Pools:                   row.Pools,
				PriceBand:               row.PriceBand,
			}
		}

		if err != nil {
			return err
		}

		// Fetch dbRuns
		loggerCtx = armadacontext.New(ctx, ctx.Logger().WithField("query", "updated-runs"))
		updatedRuns, err = fetch(loggerCtx, jobRunSerial, r.batchSize, func(from int64) ([]Run, error) {
			return queries.SelectNewRuns(ctx, SelectNewRunsParams{Serial: from, Limit: r.batchSize})
		})

		return err
	})

	return updatedJobs, updatedRuns, err
}

// FindInactiveRuns returns a slice containing all dbRuns that the scheduler does not currently consider active
// Runs are inactive if they don't exist or if they have succeeded, failed or been cancelled
func (r *PostgresJobRepository) FindInactiveRuns(ctx *armadacontext.Context, runIds []string) ([]string, error) {
	var inactiveRuns []string
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		tmpTable, err := insertRunIdsToTmpTable(ctx, tx, runIds)
		if err != nil {
			return err
		}

		query := `
		SELECT tmp.run_id
		FROM %s as tmp
		LEFT JOIN runs ON (tmp.run_id = runs.run_id)
		WHERE runs.run_id IS NULL
		OR runs.succeeded = true
 		OR runs.failed = true
		OR runs.cancelled = true;`

		rows, err := tx.Query(ctx, fmt.Sprintf(query, tmpTable))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			runId := ""
			err = rows.Scan(&runId)
			if err != nil {
				return errors.WithStack(err)
			}
			inactiveRuns = append(inactiveRuns, runId)
		}
		return nil
	})
	return inactiveRuns, err
}

// FetchJobRunLeases fetches new job runs for a given executor.  A maximum of maxResults rows will be returned, while run
// in excludedRunIds will be excluded
func (r *PostgresJobRepository) FetchJobRunLeases(ctx *armadacontext.Context, executor string, maxResults uint, excludedRunIds []string) ([]*JobRunLease, error) {
	if maxResults == 0 {
		return []*JobRunLease{}, nil
	}
	var newRuns []*JobRunLease
	err := pgx.BeginTxFunc(ctx, r.db, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		tmpTable, err := insertRunIdsToTmpTable(ctx, tx, excludedRunIds)
		if err != nil {
			return err
		}

		query := `
				SELECT jr.run_id, jr.node, j.queue, j.job_set,  jr.pool, j.user_id, j.groups, j.submit_message, jr.pod_requirements_overlay
				FROM runs jr
				LEFT JOIN %s as tmp ON (tmp.run_id = jr.run_id)
			    JOIN jobs j
			    ON jr.job_id = j.job_id
				WHERE jr.executor = $1
			    AND tmp.run_id IS NULL
				AND jr.succeeded = false
				AND jr.failed = false
				AND jr.cancelled = false
				ORDER BY jr.serial
				LIMIT %d;
`

		rows, err := tx.Query(ctx, fmt.Sprintf(query, tmpTable, maxResults), executor)
		if err != nil {
			return errors.WithStack(err)
		}
		defer rows.Close()
		for rows.Next() {
			run := JobRunLease{}
			err = rows.Scan(&run.RunID, &run.Node, &run.Queue, &run.JobSet, &run.Pool, &run.UserID, &run.Groups, &run.SubmitMessage, &run.PodRequirementsOverlay)
			if err != nil {
				return errors.WithStack(err)
			}
			newRuns = append(newRuns, &run)
		}
		return nil
	})
	return newRuns, err
}

// CountReceivedPartitions returns a count of the number of partition messages present in the database corresponding
// to the provided groupId.  This is used by the scheduler to determine if the database represents the state of
// pulsar after a given point in time.
func (r *PostgresJobRepository) CountReceivedPartitions(ctx *armadacontext.Context, groupId uuid.UUID) (uint32, error) {
	queries := New(r.db)
	count, err := queries.CountGroup(ctx, groupId)
	if err != nil {
		return 0, err
	}
	return uint32(count), nil
}

// fetch gets all rows from the database with a serial greater than from.
// Rows are fetched in batches using the supplied fetchBatch function
func fetch[T hasSerial](ctx *armadacontext.Context, from int64, batchSize int32, fetchBatch func(int64) ([]T, error)) ([]T, error) {
	timeOfLastLogging := time.Now()
	values := make([]T, 0)
	ctx.Infof("fetching in batches of size %d", batchSize)
	for {
		batch, err := fetchBatch(from)
		if err != nil {
			return nil, err
		}
		values = append(values, batch...)
		if len(batch) < int(batchSize) {
			break
		}
		from = batch[len(batch)-1].GetSerial()
		if time.Now().Sub(timeOfLastLogging) > time.Second*5 {
			ctx.Infof("fetched %d rows so far", len(values))
			timeOfLastLogging = time.Now()
		}
	}
	ctx.Infof("finished fetching - %d rows fetched", len(values))
	return values, nil
}

// Insert all run ids into a tmp table.  The name of the table is returned
func insertRunIdsToTmpTable(ctx *armadacontext.Context, tx pgx.Tx, runIds []string) (string, error) {
	tmpTable := database.UniqueTableName("job_runs")

	_, err := tx.Exec(ctx, fmt.Sprintf("CREATE TEMPORARY TABLE %s (run_id text) ON COMMIT DROP", tmpTable))
	if err != nil {
		return "", errors.WithStack(err)
	}
	_, err = tx.CopyFrom(ctx,
		pgx.Identifier{tmpTable},
		[]string{"run_id"},
		pgx.CopyFromSlice(len(runIds), func(i int) ([]interface{}, error) {
			return []interface{}{
				runIds[i],
			}, nil
		}),
	)
	return tmpTable, err
}
