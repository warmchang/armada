package lookoutdb

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/armadaproject/armada/internal/common/armadacontext"
	"github.com/armadaproject/armada/internal/common/armadaerrors"
	"github.com/armadaproject/armada/internal/common/database/lookout"
	commonmetrics "github.com/armadaproject/armada/internal/common/ingest/metrics"
	log "github.com/armadaproject/armada/internal/common/logging"
	"github.com/armadaproject/armada/internal/lookoutingester/metrics"
	"github.com/armadaproject/armada/internal/lookoutingester/model"
)

type LookoutDb struct {
	db          *pgxpool.Pool
	metrics     *metrics.Metrics
	maxBackoff  int
	fatalErrors []*regexp.Regexp
}

func NewLookoutDb(db *pgxpool.Pool, fatalErrors []*regexp.Regexp, metrics *metrics.Metrics, maxBackoff int) *LookoutDb {
	return &LookoutDb{
		db:          db,
		metrics:     metrics,
		maxBackoff:  maxBackoff,
		fatalErrors: fatalErrors,
	}
}

// Store updates the lookout database according to the supplied InstructionSet.
// The updates are applied in the following order:
// * New Job Creations
// * Job Updates, New Job Creations, New User Annotations
// * Job Run Updates
// In each case we first try to bach insert the rows using the postgres copy protocol.  If this fails then we try a
// slower, serial insert and discard any rows that cannot be inserted.
func (l *LookoutDb) Store(ctx *armadacontext.Context, instructions *model.InstructionSet) error {
	// We might have multiple updates for the same job or job run
	// These can be conflated to help performance
	jobsToUpdate := conflateJobUpdates(instructions.JobsToUpdate)
	jobRunsToUpdate := conflateJobRunUpdates(instructions.JobRunsToUpdate)

	numRowsToChange := len(instructions.JobsToCreate) + len(jobsToUpdate) + len(instructions.JobRunsToCreate) +
		len(jobRunsToUpdate) + len(instructions.JobErrorsToCreate)

	start := time.Now()
	// Jobs need to be ingested first as other updates may reference these
	wgJobIngestion := sync.WaitGroup{}
	wgJobIngestion.Add(2)
	go func() {
		defer wgJobIngestion.Done()
		l.CreateJobs(ctx, instructions.JobsToCreate)
	}()
	go func() {
		defer wgJobIngestion.Done()
		l.CreateJobSpecs(ctx, instructions.JobsToCreate)
	}()

	wgJobIngestion.Wait()

	// Now we can job updates, annotations and new job runs
	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		l.UpdateJobs(ctx, jobsToUpdate)
	}()
	go func() {
		defer wg.Done()
		l.CreateJobRuns(ctx, instructions.JobRunsToCreate)
	}()
	go func() {
		defer wg.Done()
		l.CreateJobErrors(ctx, instructions.JobErrorsToCreate)
	}()

	wg.Wait()

	// Finally, we can update the job runs
	l.UpdateJobRuns(ctx, jobRunsToUpdate)

	taken := time.Since(start)
	if numRowsToChange != 0 && taken != 0 {
		l.metrics.RecordAvRowChangeTime(numRowsToChange, taken)
	}
	return nil
}

func (l *LookoutDb) CreateJobs(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	err := l.CreateJobsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Creating jobs via batch failed, will attempt to insert serially (this might be slow).")
		l.CreateJobsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job", commonmetrics.DBOperationInsert, len(instructions), taken)
	l.metrics.RecordRowsChange("job", commonmetrics.DBOperationInsert, len(instructions))
	log.Infof("Inserted %d jobs in %s", len(instructions), taken)
}

func (l *LookoutDb) CreateJobSpecs(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	err := l.CreateJobSpecsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Creating job specs via batch failed, will attempt to insert serially (this might be slow).")
		l.CreateJobSpecsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job_spec", commonmetrics.DBOperationInsert, len(instructions), taken)
	l.metrics.RecordRowsChange("job_spec", commonmetrics.DBOperationInsert, len(instructions))
	log.Infof("Inserted %d job specs in %s", len(instructions), taken)
}

func (l *LookoutDb) UpdateJobs(ctx *armadacontext.Context, instructions []*model.UpdateJobInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	instructions = l.filterEventsForTerminalJobs(ctx, l.db, instructions, l.metrics)
	err := l.UpdateJobsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Updating jobs via batch failed, will attempt to insert serially (this might be slow).")
		l.UpdateJobsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job", commonmetrics.DBOperationUpdate, len(instructions), taken)
	l.metrics.RecordRowsChange("job", commonmetrics.DBOperationUpdate, len(instructions))
	log.Infof("Updated %d jobs in %s", len(instructions), taken)
}

func (l *LookoutDb) CreateJobRuns(ctx *armadacontext.Context, instructions []*model.CreateJobRunInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	err := l.CreateJobRunsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Creating job runs via batch failed, will attempt to insert serially (this might be slow).")
		l.CreateJobRunsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job_run", commonmetrics.DBOperationInsert, len(instructions), taken)
	l.metrics.RecordRowsChange("job_run", commonmetrics.DBOperationInsert, len(instructions))
	log.Infof("Inserted %d job runs in %s", len(instructions), taken)
}

func (l *LookoutDb) UpdateJobRuns(ctx *armadacontext.Context, instructions []*model.UpdateJobRunInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	err := l.UpdateJobRunsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Updating job runs via batch failed, will attempt to insert serially (this might be slow).")
		l.UpdateJobRunsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job_run", commonmetrics.DBOperationUpdate, len(instructions), taken)
	l.metrics.RecordRowsChange("job_run", commonmetrics.DBOperationUpdate, len(instructions))
	log.Infof("Updated %d job runs in %s", len(instructions), taken)
}

func (l *LookoutDb) CreateJobErrors(ctx *armadacontext.Context, instructions []*model.CreateJobErrorInstruction) {
	if len(instructions) == 0 {
		return
	}
	start := time.Now()
	err := l.CreateJobErrorsBatch(ctx, instructions)
	if err != nil {
		log.WithError(err).Warn("Creating job errors via batch failed, will attempt to insert serially (this might be slow).")
		l.CreateJobErrorsScalar(ctx, instructions)
	}
	taken := time.Since(start)
	l.metrics.RecordAvRowChangeTimeByOperation("job_error", commonmetrics.DBOperationInsert, len(instructions), taken)
	l.metrics.RecordRowsChange("job_error", commonmetrics.DBOperationInsert, len(instructions))
	log.Infof("Inserted %d job errors in %s", len(instructions), taken)
}

func (l *LookoutDb) CreateJobsBatch(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) error {
	return l.withDatabaseRetryInsert(func() error {
		tmpTable := "job_create_tmp"

		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s
				(
					job_id 	                     varchar(32),
					queue                        varchar(512),
					owner                        varchar(512),
					namespace                    varchar(512),
					jobset                       varchar(1024),
					cpu                          bigint,
					memory                       bigint,
					ephemeral_storage            bigint,
					gpu                          bigint,
					priority                     bigint,
					submitted                    timestamp,
					state                        smallint,
					last_transition_time         timestamp,
					last_transition_time_seconds bigint,
					priority_class               varchar(63),
					annotations                  jsonb,
				    external_job_uri			 varchar(1024) NULL
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"job_id",
					"queue",
					"owner",
					"namespace",
					"jobset",
					"cpu",
					"memory",
					"ephemeral_storage",
					"gpu",
					"priority",
					"submitted",
					"state",
					"last_transition_time",
					"last_transition_time_seconds",
					"priority_class",
					"annotations",
					"external_job_uri",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].JobId,
						instructions[i].Queue,
						instructions[i].Owner,
						instructions[i].Namespace,
						instructions[i].JobSet,
						instructions[i].Cpu,
						instructions[i].Memory,
						instructions[i].EphemeralStorage,
						instructions[i].Gpu,
						instructions[i].Priority,
						instructions[i].Submitted,
						instructions[i].State,
						instructions[i].LastTransitionTime,
						instructions[i].LastTransitionTimeSeconds,
						instructions[i].PriorityClass,
						instructions[i].Annotations,
						instructions[i].ExternalJobUri,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`
					INSERT INTO job (
						job_id,
						queue,
						owner,
						namespace,
						jobset,
						cpu,
						memory,
						ephemeral_storage,
						gpu,
						priority,
						submitted,
						state,
						last_transition_time,
						last_transition_time_seconds,
						priority_class,
						annotations,
					    external_job_uri
					) SELECT * from %s
					ON CONFLICT DO NOTHING`, tmpTable),
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		}

		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

// CreateJobsScalar will insert jobs one by one into the database
func (l *LookoutDb) CreateJobsScalar(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) {
	sqlStatement := `INSERT INTO job (
			job_id,
			queue,
			owner,
			namespace,
			jobset,
			cpu,
			memory,
			ephemeral_storage,
			gpu,
			priority,
			submitted,
			state,
			last_transition_time,
			last_transition_time_seconds,
			priority_class,
			annotations,
            external_job_uri
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT DO NOTHING`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.JobId,
				i.Queue,
				i.Owner,
				i.Namespace,
				i.JobSet,
				i.Cpu,
				i.Memory,
				i.EphemeralStorage,
				i.Gpu,
				i.Priority,
				i.Submitted,
				i.State,
				i.LastTransitionTime,
				i.LastTransitionTimeSeconds,
				i.PriorityClass,
				i.Annotations,
				i.ExternalJobUri,
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Create job for job %s, jobset %s failed", i.JobId, i.JobSet)
		}
	}
}

func (l *LookoutDb) UpdateJobsBatch(ctx *armadacontext.Context, instructions []*model.UpdateJobInstruction) error {
	return l.withDatabaseRetryInsert(func() error {
		tmpTable := "job_update_tmp"

		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s (
					job_id                       varchar(32),
					priority                     bigint,
					state                        smallint,
					cancelled                    timestamp,
					last_transition_time         timestamp,
					last_transition_time_seconds bigint,
					duplicate                    bool,
					latest_run_id                varchar(36),
					cancel_reason                varchar(512),
					cancel_user                  varchar(512)
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"job_id",
					"priority",
					"state",
					"cancelled",
					"last_transition_time",
					"last_transition_time_seconds",
					"duplicate",
					"latest_run_id",
					"cancel_reason",
					"cancel_user",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].JobId,
						instructions[i].Priority,
						instructions[i].State,
						instructions[i].Cancelled,
						instructions[i].LastTransitionTime,
						instructions[i].LastTransitionTimeSeconds,
						instructions[i].Duplicate,
						instructions[i].LatestRunId,
						instructions[i].CancelReason,
						instructions[i].CancelUser,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`UPDATE job
					SET
						priority                     = coalesce(tmp.priority, job.priority),
						state                        = coalesce(tmp.state, job.state),
						cancelled                    = coalesce(tmp.cancelled, job.cancelled),
						last_transition_time         = coalesce(tmp.last_transition_time, job.last_transition_time),
						last_transition_time_seconds = coalesce(tmp.last_transition_time_seconds, job.last_transition_time_seconds),
						duplicate                    = coalesce(tmp.duplicate, job.duplicate),
						latest_run_id                = coalesce(tmp.latest_run_id, job.latest_run_id),
						cancel_reason                = coalesce(tmp.cancel_reason, job.cancel_reason),
						cancel_user                  = coalesce(tmp.cancel_user, job.cancel_user)
					FROM %s as tmp WHERE tmp.job_id = job.job_id`, tmpTable),
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationUpdate)
			}
			return err
		}

		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

func (l *LookoutDb) UpdateJobsScalar(ctx *armadacontext.Context, instructions []*model.UpdateJobInstruction) {
	sqlStatement := `UPDATE job
		SET
			priority                     = coalesce($2, priority),
			state                        = coalesce($3, state),
			cancelled                    = coalesce($4, cancelled),
			last_transition_time         = coalesce($5, job.last_transition_time),
			last_transition_time_seconds = coalesce($6, job.last_transition_time_seconds),
			duplicate                    = coalesce($7, duplicate),
			latest_run_id                = coalesce($8, job.latest_run_id),
			cancel_reason                = coalesce($9, job.cancel_reason),
			cancel_user                  = coalesce($10, job.cancel_user)
		WHERE job_id = $1`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.JobId,
				i.Priority,
				i.State,
				i.Cancelled,
				i.LastTransitionTime,
				i.LastTransitionTimeSeconds,
				i.Duplicate,
				i.LatestRunId,
				i.CancelReason,
				i.CancelUser)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationUpdate)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Updating job %s failed", i.JobId)
		}
	}
}

func (l *LookoutDb) CreateJobSpecsBatch(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) error {
	return l.withDatabaseRetryInsert(func() error {
		tmpTable := "job_spec_create_tmp"

		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s (
					job_id varchar(32),
					job_spec bytea
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"job_id",
					"job_spec",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].JobId,
						instructions[i].JobProto,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`
					INSERT INTO job_spec (
						job_id,
						job_spec
					) SELECT * from %s
					ON CONFLICT DO NOTHING`, tmpTable),
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		}

		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

func (l *LookoutDb) CreateJobSpecsScalar(ctx *armadacontext.Context, instructions []*model.CreateJobInstruction) {
	sqlStatement := `INSERT INTO job_spec (
			job_id,
			job_spec
		)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.JobId,
				i.JobProto,
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Create job spec for job %s, jobset %s failed", i.JobId, i.JobSet)
		}
	}
}

func (l *LookoutDb) CreateJobRunsBatch(ctx *armadacontext.Context, instructions []*model.CreateJobRunInstruction) error {
	return l.withDatabaseRetryInsert(func() error {
		tmpTable := "job_run_create_tmp"

		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s (
					run_id        varchar(36),
					job_id        varchar(32),
					cluster       varchar(512),
					node          varchar(512),
					leased        timestamp,
					pending       timestamp,
					job_run_state smallint,
					pool 		  text
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"run_id",
					"job_id",
					"cluster",
					"node",
					"leased",
					"pending",
					"job_run_state",
					"pool",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].RunId,
						instructions[i].JobId,
						instructions[i].Cluster,
						instructions[i].Node,
						instructions[i].Leased,
						instructions[i].Pending,
						instructions[i].JobRunState,
						instructions[i].Pool,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`
					INSERT INTO job_run (
						run_id,
						job_id,
						cluster,
						node,
						leased,
						pending,
						job_run_state,
						pool
					) SELECT * from %s
					ON CONFLICT DO NOTHING`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		}
		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

func (l *LookoutDb) CreateJobRunsScalar(ctx *armadacontext.Context, instructions []*model.CreateJobRunInstruction) {
	sqlStatement := `INSERT INTO job_run (
			run_id,
			job_id,
			cluster,
			node,
			leased,
			pending,
			job_run_state,
			pool)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT DO NOTHING`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.RunId,
				i.JobId,
				i.Cluster,
				i.Node,
				i.Leased,
				i.Pending,
				i.JobRunState,
				i.Pool)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Create job run for job %s, run %s failed", i.JobId, i.RunId)
		}
	}
}

func (l *LookoutDb) UpdateJobRunsBatch(ctx *armadacontext.Context, instructions []*model.UpdateJobRunInstruction) error {
	return l.withDatabaseRetryInsert(func() error {
		tmpTable := "job_run_update_tmp"

		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s (
					run_id        varchar(36),
					node          varchar(512),
				    pending       timestamp,
					started       timestamp,
					finished      timestamp,
				    job_run_state smallint,
					error         bytea,
					debug         bytea,
				    exit_code     int
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"run_id",
					"node",
					"pending",
					"started",
					"finished",
					"job_run_state",
					"error",
					"debug",
					"exit_code",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].RunId,
						instructions[i].Node,
						instructions[i].Pending,
						instructions[i].Started,
						instructions[i].Finished,
						instructions[i].JobRunState,
						instructions[i].Error,
						instructions[i].Debug,
						instructions[i].ExitCode,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`UPDATE job_run
					SET
						node          = coalesce(tmp.node, job_run.node),
						pending       = coalesce(tmp.pending, job_run.pending),
						started       = coalesce(tmp.started, job_run.started),
						finished      = coalesce(tmp.finished, job_run.finished),
						job_run_state = coalesce(tmp.job_run_state, job_run.job_run_state),
						error         = coalesce(tmp.error, job_run.error),
						debug         = coalesce(tmp.debug, job_run.debug),
						exit_code     = coalesce(tmp.exit_code, job_run.exit_code)
					FROM %s as tmp where tmp.run_id = job_run.run_id`, tmpTable),
			)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationUpdate)
			}
			return err
		}

		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

func (l *LookoutDb) UpdateJobRunsScalar(ctx *armadacontext.Context, instructions []*model.UpdateJobRunInstruction) {
	sqlStatement := `UPDATE job_run
		SET
			node          = coalesce($2, node),
			started       = coalesce($3, started),
			finished      = coalesce($4, finished),
			job_run_state = coalesce($5, job_run_state),
			error         = coalesce($6, error),
			exit_code     = coalesce($7, exit_code),
			pending       = coalesce($8, pending),
			debug         = coalesce($9, debug)
		WHERE run_id = $1`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.RunId,
				i.Node,
				i.Started,
				i.Finished,
				i.JobRunState,
				i.Error,
				i.ExitCode,
				i.Pending,
				i.Debug)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationUpdate)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Updating job run %s failed", i.RunId)
		}
	}
}

func (l *LookoutDb) CreateJobErrorsBatch(ctx *armadacontext.Context, instructions []*model.CreateJobErrorInstruction) error {
	tmpTable := "job_error_create_tmp"
	return l.withDatabaseRetryInsert(func() error {
		createTmp := func(tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf(`
				CREATE TEMPORARY TABLE %s (
					job_id varchar(32),
					error bytea
				) ON COMMIT DROP;`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationCreateTempTable)
			}
			return err
		}

		insertTmp := func(tx pgx.Tx) error {
			_, err := tx.CopyFrom(ctx,
				pgx.Identifier{tmpTable},
				[]string{
					"job_id",
					"error",
				},
				pgx.CopyFromSlice(len(instructions), func(i int) ([]interface{}, error) {
					return []interface{}{
						instructions[i].JobId,
						instructions[i].Error,
					}, nil
				}),
			)
			return err
		}

		copyToDest := func(tx pgx.Tx) error {
			_, err := tx.Exec(
				ctx,
				fmt.Sprintf(`
					INSERT INTO job_error (
						job_id,
						error
					) SELECT * from %s
					ON CONFLICT DO NOTHING`, tmpTable))
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		}
		return batchInsert(ctx, l.db, createTmp, insertTmp, copyToDest)
	})
}

func (l *LookoutDb) CreateJobErrorsScalar(ctx *armadacontext.Context, instructions []*model.CreateJobErrorInstruction) {
	sqlStatement := `INSERT INTO job_error (job_id, error)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`
	for _, i := range instructions {
		err := l.withDatabaseRetryInsert(func() error {
			_, err := l.db.Exec(ctx, sqlStatement,
				i.JobId,
				i.Error)
			if err != nil {
				l.metrics.RecordDBError(commonmetrics.DBOperationInsert)
			}
			return err
		})
		if err != nil {
			log.WithError(err).Warnf("Create job error for job %s, failed", i.JobId)
		}
	}
}

func batchInsert(ctx *armadacontext.Context, db *pgxpool.Pool, createTmp func(pgx.Tx) error,
	insertTmp func(pgx.Tx) error, copyToDest func(pgx.Tx) error,
) error {
	return pgx.BeginTxFunc(ctx, db, pgx.TxOptions{
		IsoLevel:       pgx.ReadCommitted,
		AccessMode:     pgx.ReadWrite,
		DeferrableMode: pgx.Deferrable,
	}, func(tx pgx.Tx) error {
		// Create a temporary table to hold the staging data
		err := createTmp(tx)
		if err != nil {
			return err
		}

		err = insertTmp(tx)
		if err != nil {
			return err
		}

		err = copyToDest(tx)
		if err != nil {
			return err
		}
		return nil
	})
}

func conflateJobUpdates(updates []*model.UpdateJobInstruction) []*model.UpdateJobInstruction {
	isTerminal := func(p *int32) bool {
		if p == nil {
			return false
		} else {
			return *p == lookout.JobFailedOrdinal ||
				*p == lookout.JobSucceededOrdinal ||
				*p == lookout.JobCancelledOrdinal ||
				*p == lookout.JobPreemptedOrdinal ||
				*p == lookout.JobRejectedOrdinal
		}
	}

	updatesById := make(map[string]*model.UpdateJobInstruction)
	for _, update := range updates {
		existing, ok := updatesById[update.JobId]
		if !ok {
			updatesById[update.JobId] = update
			continue
		}

		// Unfortunately once a job has reached a terminal state we still get state updates for it e.g. we can get an event to
		// say it's now "running".  We have to throw these away else we'll end up with a zombie job.
		//
		// Need to deal with preempted jobs separately
		// We could get a JobFailed event (or another terminal event) after a JobPreempted event,
		// but the job should still be marked as Preempted
		if !isTerminal(existing.State) || (update.State != nil && *update.State == lookout.JobPreemptedOrdinal) {
			if update.Priority != nil {
				existing.Priority = update.Priority
			}
			if update.State != nil {
				existing.State = update.State
			}
			if update.Cancelled != nil {
				existing.Cancelled = update.Cancelled
			}
			if update.LastTransitionTime != nil {
				existing.LastTransitionTime = update.LastTransitionTime
			}
			if update.LastTransitionTimeSeconds != nil {
				existing.LastTransitionTimeSeconds = update.LastTransitionTimeSeconds
			}
			if update.Duplicate != nil {
				existing.Duplicate = update.Duplicate
			}
			if update.LatestRunId != nil {
				existing.LatestRunId = update.LatestRunId
			}
		}
	}

	conflated := make([]*model.UpdateJobInstruction, 0, len(updatesById))
	// TODO: it turns out that iteration over a map in go yields a different key order each time!
	// This means that that slice outputted by this function (and the one below) will have a random order
	// This isn't a problem as such for the database but does mean that reproducing errors etc will be hard
	for _, v := range updatesById {
		conflated = append(conflated, v)
	}
	return conflated
}

func conflateJobRunUpdates(updates []*model.UpdateJobRunInstruction) []*model.UpdateJobRunInstruction {
	updatesById := make(map[string]*model.UpdateJobRunInstruction)
	for _, update := range updates {
		existing, ok := updatesById[update.RunId]
		if ok {
			if update.Node != nil {
				existing.Node = update.Node
			}
			if update.Started != nil {
				existing.Started = update.Started
			}
			if update.Finished != nil {
				existing.Finished = update.Finished
			}
			if update.Error != nil {
				existing.Error = update.Error
			}
			if update.Debug != nil {
				existing.Debug = update.Debug
			}
			if update.JobRunState != nil {
				existing.JobRunState = update.JobRunState
			}
			if update.ExitCode != nil {
				existing.ExitCode = update.ExitCode
			}
		} else {
			updatesById[update.RunId] = update
		}
	}

	conflated := make([]*model.UpdateJobRunInstruction, 0, len(updatesById))
	for _, v := range updatesById {
		conflated = append(conflated, v)
	}
	return conflated
}

// updateInstructionsForJob is used in filterEventsForTerminalJobs, and records a list of job updates for a single job,
// along with whether it contains an instruction corresponding to a JobPreempted event
type updateInstructionsForJob struct {
	instructions      []*model.UpdateJobInstruction
	containsPreempted bool
}

// filterEventsForTerminalJobs queries the database for any jobs that are in a terminal state and removes them from the list of
// instructions.  This is necessary because Armada will generate event statuses even for jobs that have reached a terminal state
// The proper solution here is to make it so once a job is terminal, no more events are generated for it, but until
// that day we have to manually filter them out here.
// NOTE: this function will retry querying the database for as long as possible in order to determine which jobs are
// in the terminal state.  If, however, the database returns a non-retryable error it will give up and simply not
// filter out any events as the job state is undetermined.
func (l *LookoutDb) filterEventsForTerminalJobs(
	ctx *armadacontext.Context,
	db *pgxpool.Pool,
	instructions []*model.UpdateJobInstruction,
	m *metrics.Metrics,
) []*model.UpdateJobInstruction {
	jobIds := make([]string, len(instructions))
	for i, instruction := range instructions {
		jobIds[i] = instruction.JobId
	}
	queryStart := time.Now()
	rowsRaw, err := l.withDatabaseRetryQuery(func() (interface{}, error) {
		terminalStates := []int{
			lookout.JobSucceededOrdinal,
			lookout.JobFailedOrdinal,
			lookout.JobCancelledOrdinal,
			lookout.JobPreemptedOrdinal,
			lookout.JobRejectedOrdinal,
		}
		return db.Query(ctx, "SELECT DISTINCT job_id, state FROM JOB where state = any($1) AND job_id = any($2)", terminalStates, jobIds)
	})
	if err != nil {
		m.RecordDBError(commonmetrics.DBOperationRead)
		log.WithError(err).Warnf("Cannot retrieve job state from the database- Cancelled jobs may not be filtered out")
		return instructions
	}
	rows := rowsRaw.(pgx.Rows)

	terminalJobs := make(map[string]int)
	for rows.Next() {
		jobId := ""
		var state int16
		err := rows.Scan(&jobId, &state)
		if err != nil {
			log.WithError(err).Warnf("Cannot retrieve jobId from row. Terminal jobs will not be filtered out")
		} else {
			terminalJobs[jobId] = int(state)
		}
	}
	log.Infof("Lookup of terminal states for %d jobs took %s and returned  %d results", len(instructions), time.Since(queryStart), len(terminalJobs))

	if len(terminalJobs) > 0 {
		jobInstructionMap := make(map[string]*updateInstructionsForJob)
		for _, instruction := range instructions {
			data, ok := jobInstructionMap[instruction.JobId]
			if !ok {
				data = &updateInstructionsForJob{
					instructions:      []*model.UpdateJobInstruction{},
					containsPreempted: false,
				}
				jobInstructionMap[instruction.JobId] = data
			}
			data.instructions = append(data.instructions, instruction)
			data.containsPreempted = instruction.State != nil && *instruction.State == lookout.JobPreemptedOrdinal
		}

		var filtered []*model.UpdateJobInstruction
		for jobId, updateInstructions := range jobInstructionMap {
			state, ok := terminalJobs[jobId]
			// Need to record updates if either:
			// * Job is not in a terminal state
			// * Job is in succeeded, failed or cancelled, but job preempted event will be recorded
			if !ok || ((state == lookout.JobSucceededOrdinal ||
				state == lookout.JobFailedOrdinal ||
				state == lookout.JobCancelledOrdinal) && updateInstructions.containsPreempted) {
				filtered = append(filtered, updateInstructions.instructions...)
			}
		}
		return filtered
	} else {
		return instructions
	}
}

func (l *LookoutDb) withDatabaseRetryInsert(executeDb func() error) error {
	_, err := l.withDatabaseRetryQuery(func() (interface{}, error) {
		return nil, executeDb()
	})
	return err
}

// Executes a database function, retrying until it either succeeds or encounters a non-retryable error
func (l *LookoutDb) withDatabaseRetryQuery(executeDb func() (interface{}, error)) (interface{}, error) {
	// TODO: arguably this should come from config
	backOff := 1
	for {
		res, err := executeDb()

		if err == nil {
			return res, nil
		}

		if armadaerrors.IsRetryablePostgresError(err, l.fatalErrors) {
			backOff = min(2*backOff, l.maxBackoff)
			log.WithError(err).Warnf("Retryable error encountered executing sql, will wait for %d seconds before retrying.", backOff)
			time.Sleep(time.Duration(backOff) * time.Second)
		} else {
			// Non retryable error
			return nil, err
		}
	}
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
