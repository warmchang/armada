package performance

import (
	"context"
	"fmt"
	"github.com/G-Research/armada/internal/common/compress"
	"github.com/G-Research/armada/internal/common/util"
	"github.com/G-Research/armada/pkg/api"
	"github.com/gogo/protobuf/proto"
	"github.com/google/uuid"
	"github.com/jackc/pgtype/pgxtype"
	"github.com/jackc/pgx/v4"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"math/rand"
	"strings"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/G-Research/armada/internal/armada/configuration"
	"github.com/G-Research/armada/internal/lookout/postgres"
)

const (
	Pending   = 0
	Running   = 1
	Succeeded = 2
	Failed    = 3
)

var SampleError = strings.Repeat("abcdefghijklmnop", 128)

var SampleBadExitCode = 137
var SampleGoodExitCode = 0

var SmallSpec = CreateJobsSpec{
	Queues:                          5,
	JobSetsPerQueue:                 10,
	JobsPerJobSet:                   5,
	Clusters:                        2,
	NodesPerCluster:                 2,
	Annotations:                     3,
	AnnotationValuesPerKey:          5,
	AnnotationsPerJob:               1,
	JobArgsSize:                     10,
	JobsWithMoreThanOneRun:          50,
	RunsPerJobWithMoreThanOneJobRun: 3,
}

var VgisSpec = CreateJobsSpec{
	Queues:                          400,
	JobSetsPerQueue:                 2000,
	JobsPerJobSet:                   7,
	Clusters:                        6,
	NodesPerCluster:                 30,
	Annotations:                     61,
	AnnotationValuesPerKey:          10000,
	AnnotationsPerJob:               10,
	JobArgsSize:                     32768,
	JobsWithMoreThanOneRun:          420000,
	RunsPerJobWithMoreThanOneJobRun: 3,
}

type LookoutJobRun struct {
	RunId       string
	JobId       string
	PodNumber   int
	Cluster     string
	Node        string
	Leased      time.Time
	Started     time.Time
	Finished    time.Time
	JobRunState int
	Error       *string
	ExitCode    *int
	Active      bool
}

type LookoutJobResources struct {
	Cpu              int64
	Memory           int64
	EphemeralStorage int64
	Gpu              int64
}

type LookoutJob struct {
	Job                *api.Job
	Runs               []*LookoutJobRun
	Submitted          time.Time
	Cancelled          time.Time
	State              int
	LastTransitionTime time.Time
	Duplicate          bool
	Resources          *LookoutJobResources
}

type LookoutAnnotation struct {
	JobId string
	Key   string
	Value string
}

type CreateJobsSpec struct {
	Queues                          int
	JobSetsPerQueue                 int
	JobsPerJobSet                   int
	Clusters                        int
	NodesPerCluster                 int
	Annotations                     int
	AnnotationValuesPerKey          int
	AnnotationsPerJob               int
	JobArgsSize                     int
	JobsWithMoreThanOneRun          int
	RunsPerJobWithMoreThanOneJobRun int
}

type QueryTest struct {
	Name  string
	Query string
	Args  []interface{}
}

func PanicOnErr(err error) {
	if err != nil {
		panic(err)
	}
}

func OpenDb() *pgxpool.Pool {
	postgresConfig := configuration.PostgresConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    10,
		ConnMaxLifetime: 10 * time.Minute,
		Connection: map[string]string{
			"host":     "localhost",
			"port":     "10000",
			"user":     "postgres",
			"password": "psw",
			"dbname":   "postgres",
			"sslmode":  "disable",
		},
	}

	db, err := postgres.OpenPgxPool(postgresConfig)
	PanicOnErr(err)
	return db
}

func ExecFileDb(db pgxtype.Querier, filepath string) {
	c, err := ioutil.ReadFile(filepath)
	PanicOnErr(err)
	sql := string(c)

	_, err = db.Exec(context.TODO(), sql)
	PanicOnErr(err)
}

func CreateRuns(idx int, nRuns int, jobId string, allClusters []string, clusterNodeMapping map[string][]string, needsActiveRun bool) []*LookoutJobRun {
	runs := []*LookoutJobRun{}
	for i := 0; i < nRuns; i++ {
		cluster := allClusters[(idx+i)%len(allClusters)]
		var err *string = nil
		var exitCode *int = nil
		if (idx+i)%2 == 0 {
			exitCode = &SampleGoodExitCode
		}
		if (idx+i)%5 == 0 {
			err = &SampleError
			exitCode = &SampleBadExitCode
		}
		runs = append(runs, &LookoutJobRun{
			RunId:       uuid.NewString(),
			JobId:       jobId,
			PodNumber:   0,
			Cluster:     cluster,
			Node:        clusterNodeMapping[cluster][(idx+i)%len(clusterNodeMapping[cluster])],
			Leased:      time.Now(),
			Started:     time.Now(),
			Finished:    time.Now(),
			JobRunState: (idx + i) % 4,
			Error:       err,
			ExitCode:    exitCode,
			Active:      i == 0 && needsActiveRun,
		})
	}
	return runs
}

func CreateJobs(spec CreateJobsSpec) []*LookoutJob {
	jobs := []*LookoutJob{}

	jobArgs := strings.Repeat("x", spec.JobArgsSize)

	allClusters := make([]string, spec.Clusters, spec.Clusters)
	clusterNodeMapping := map[string][]string{}
	for i := 0; i < spec.Clusters; i++ {
		clusterName := uuid.NewString()
		nodes := make([]string, spec.NodesPerCluster, spec.NodesPerCluster)
		for j := 0; j < spec.NodesPerCluster; j++ {
			nodes[j] = uuid.NewString()
		}
		clusterNodeMapping[clusterName] = nodes
		allClusters[i] = clusterName
	}

	allAnnotationKeys := make([]string, spec.Annotations, spec.Annotations)
	allAnnotations := map[string][]string{}
	for i := 0; i < spec.Annotations; i++ {
		key := uuid.NewString()
		annotationValues := make([]string, spec.AnnotationValuesPerKey, spec.AnnotationValuesPerKey)
		for j := 0; j < spec.AnnotationValuesPerKey; j++ {
			annotationValues[j] = uuid.NewString()
		}
		allAnnotations[key] = annotationValues
		allAnnotationKeys[i] = key
	}

	allResources := []*LookoutJobResources{}
	for i := 0; i < 10; i++ {
		allResources = append(allResources, &LookoutJobResources{
			Cpu:              (1 + rand.Int63n(8)) * 1000000000,
			Memory:           (1 + rand.Int63n(8)) * 1073741824,
			EphemeralStorage: (1 + rand.Int63n(8)) * 1073741824,
			Gpu:              1 + rand.Int63n(8),
		})
	}

	total := spec.Queues * spec.JobSetsPerQueue * spec.JobsPerJobSet
	logInterval := total / 10

	multipleRunsInterval := total / spec.JobsWithMoreThanOneRun

	for i := 0; i < spec.Queues; i++ {
		queue := uuid.NewString()
		for j := 0; j < spec.JobSetsPerQueue; j++ {
			jobSet := uuid.NewString()
			for k := 0; k < spec.JobsPerJobSet; k++ {
				jobId := util.NewULID()
				idx := i*spec.JobSetsPerQueue*spec.JobsPerJobSet + j*spec.JobsPerJobSet + k

				annotations := map[string]string{}
				for l := 0; l < spec.AnnotationsPerJob; l++ {
					key := allAnnotationKeys[(idx+l)%len(allAnnotationKeys)]
					annotations[key] = allAnnotations[key][(idx+l)%len(allAnnotations[key])]
				}

				nRuns := 1
				if idx%multipleRunsInterval == 0 {
					nRuns = spec.RunsPerJobWithMoreThanOneJobRun
				}

				job := &LookoutJob{
					Job: &api.Job{
						Id:                                 jobId,
						ClientId:                           jobId,
						JobSetId:                           jobSet,
						Queue:                              queue,
						Namespace:                          queue,
						Annotations:                        annotations,
						Owner:                              queue,
						QueueOwnershipUserGroups:           nil,
						CompressedQueueOwnershipUserGroups: nil,
						Priority:                           0,
						PodSpecs: []*v1.PodSpec{
							{
								Containers: []v1.Container{
									{
										Name:  uuid.NewString(),
										Image: uuid.NewString(),
										Args: []string{
											jobArgs,
										},
									},
								},
							},
						},
						Created: time.Time{},
					},
					Runs:               CreateRuns(idx, nRuns, jobId, allClusters, clusterNodeMapping, i%4 == 2),
					Submitted:          time.Now(),
					Cancelled:          time.Now(),
					State:              idx % 4,
					LastTransitionTime: time.Now(),
					Duplicate:          false,
					Resources:          allResources[idx%len(allResources)],
				}
				jobs = append(jobs, job)

				if logInterval == 0 || (idx+1)%logInterval == 0 {
					fmt.Printf("%d/%d\n", idx+1, total)
				}
			}
		}
	}

	return jobs
}

func Batch[T any](arr []T, batchSize int) [][]T {
	nBatches := len(arr) / batchSize
	if len(arr)%batchSize != 0 {
		nBatches++
	}
	batches := [][]T{}
	for i := 0; i < nBatches; i++ {
		var batch []T
		if i*batchSize+batchSize <= len(arr) {
			batch = arr[i*batchSize : i*batchSize+batchSize]
		} else {
			batch = arr[i*batchSize:]
		}
		batches = append(batches, batch)
	}

	return batches
}

func UniqueTableName(table string) string {
	suffix := strings.ReplaceAll(uuid.New().String(), "-", "")
	return fmt.Sprintf("%s_tmp_%s", table, suffix)
}

func ReadVersion(db pgxtype.Querier) (int, error) {
	_, err := db.Exec(context.TODO(),
		`CREATE SEQUENCE IF NOT EXISTS database_version START WITH 0 MINVALUE 0;`)
	if err != nil {
		return 0, err
	}

	result, err := db.Query(context.TODO(),
		`SELECT last_value FROM database_version`)
	if err != nil {
		return 0, err
	}

	var version int
	result.Next()
	err = result.Scan(&version)

	return version, err
}

func SetVersion(db pgxtype.Querier, version int) error {
	_, err := db.Exec(context.TODO(), `SELECT setval('database_version', $1)`, version)
	return err
}

func MigrateDatabase(db *pgxpool.Pool, migrations []string) {
	currentVersion, err := ReadVersion(db)
	if err != nil {
		panic(err)
	}
	for i, file := range migrations {
		newVersion := i + 1
		if newVersion > currentVersion {
			tx, err := db.BeginTx(context.TODO(), pgx.TxOptions{})
			PanicOnErr(err)

			ExecFileDb(tx, file)

			err = SetVersion(tx, newVersion)
			PanicOnErr(err)

			err = tx.Commit(context.TODO())
			PanicOnErr(err)
		}
		fmt.Println(file)
	}
}

func SaveJobs(tx pgx.Tx, jobs []*LookoutJob, compressor compress.Compressor) error {
	tmpTable := UniqueTableName("job")

	_, err := tx.Exec(context.TODO(), fmt.Sprintf(`
		CREATE TEMPORARY TABLE %s 
		(
		    job_id 	          varchar(32),
		    queue             varchar(512),
		    owner             varchar(512),
		    jobset            varchar(1024),
		    cpu               bigint,
		    memory            bigint,
		    ephemeral_storage bigint,
		    gpu               bigint,
		    priority          double precision,
		    submitted         timestamp,
		    cancelled         timestamp,
		    job_spec          bytea,
		    state             smallint,
		    last_transition_time timestamp,
		    duplicate         boolean
		) ON COMMIT DROP
	`, tmpTable))
	if err != nil {
		return err
	}

	_, err = tx.CopyFrom(context.TODO(),
		pgx.Identifier{tmpTable},
		[]string{
			"job_id",
			"queue",
			"owner",
			"jobset",
			"cpu",
			"memory",
			"ephemeral_storage",
			"gpu",
			"priority",
			"submitted",
			"cancelled",
			"job_spec",
			"state",
			"last_transition_time",
			"duplicate",
		},
		pgx.CopyFromSlice(len(jobs), func(i int) ([]interface{}, error) {
			jobProtoUncompressed, err := proto.Marshal(jobs[i].Job)
			if err != nil {
				return nil, err
			}
			jobProto, err := compressor.Compress(jobProtoUncompressed)
			if err != nil {
				return nil, err
			}
			return []interface{}{
				jobs[i].Job.Id,
				jobs[i].Job.Queue,
				jobs[i].Job.Owner,
				jobs[i].Job.JobSetId,
				jobs[i].Resources.Cpu,
				jobs[i].Resources.Memory,
				jobs[i].Resources.EphemeralStorage,
				jobs[i].Resources.Gpu,
				jobs[i].Job.Priority,
				jobs[i].Submitted,
				jobs[i].Cancelled,
				jobProto,
				jobs[i].State,
				jobs[i].LastTransitionTime,
				jobs[i].Duplicate,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(),
		fmt.Sprintf(`
			INSERT INTO job (
			    job_id,
			    queue,
			    owner,
			    jobset,
			    cpu,
			    memory,
			    ephemeral_storage,
			    gpu,
			    priority,
			    submitted,
			    cancelled,
			    job_spec,
			    state,
			    last_transition_time,
			    duplicate
			) SELECT * from %s
			ON CONFLICT DO NOTHING
		`, tmpTable),
	)
	if err != nil {
		return err
	}

	return err
}

func SaveJobRuns(tx pgx.Tx, jobs []*LookoutJob) error {
	tmpTable := UniqueTableName("job_run")

	jobRuns := []*LookoutJobRun{}
	for _, job := range jobs {
		jobRuns = append(jobRuns, job.Runs...)
	}

	_, err := tx.Exec(context.TODO(), fmt.Sprintf(`
		CREATE TEMPORARY TABLE %s 
		(
		    run_id 	          varchar(36),
		    job_id            varchar(32),
		    pod_number        int,
		    cluster           varchar(512),
		    node              varchar(512),
			leased            timestamp,
		    started           timestamp NULL,
		    finished          timestamp NULL,
		    job_run_state     smallint,
		    error             bytea NULL,
		    exit_code         int NULL,
		    active            boolean
		) ON COMMIT DROP
	`, tmpTable))
	if err != nil {
		return err
	}

	_, err = tx.CopyFrom(context.TODO(),
		pgx.Identifier{tmpTable},
		[]string{
			"run_id",
			"job_id",
			"pod_number",
			"cluster",
			"node",
			"leased",
			"started",
			"finished",
			"job_run_state",
			"error",
			"exit_code",
			"active",
		},
		pgx.CopyFromSlice(len(jobRuns), func(i int) ([]interface{}, error) {
			return []interface{}{
				jobRuns[i].RunId,
				jobRuns[i].JobId,
				jobRuns[i].PodNumber,
				jobRuns[i].Cluster,
				jobRuns[i].Node,
				jobRuns[i].Leased,
				jobRuns[i].Started,
				jobRuns[i].Finished,
				jobRuns[i].JobRunState,
				jobRuns[i].Error,
				jobRuns[i].ExitCode,
				jobRuns[i].Active,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(),
		fmt.Sprintf(`
			INSERT INTO job_run (
			    run_id,
				job_id,
				pod_number,
				cluster,
				node,
				leased,
				started,
				finished,
				job_run_state,
				error,
				exit_code,
				active
			) SELECT * from %s
			ON CONFLICT DO NOTHING
		`, tmpTable),
	)
	if err != nil {
		return err
	}

	return err
}

func SaveJobAnnotations(tx pgx.Tx, jobs []*LookoutJob) error {
	tmpTable := UniqueTableName("annotations")

	annotations := []*LookoutAnnotation{}
	for _, job := range jobs {
		for key, value := range job.Job.Annotations {
			annotations = append(annotations, &LookoutAnnotation{
				JobId: job.Job.Id,
				Key:   key,
				Value: value,
			})
		}
	}

	_, err := tx.Exec(context.TODO(), fmt.Sprintf(`
		CREATE TEMPORARY TABLE %s 
		(
		    job_id varchar(32),
		    key    varchar(1024),
		    value  varchar(1024)
		) ON COMMIT DROP;
	`, tmpTable))
	if err != nil {
		return err
	}

	_, err = tx.CopyFrom(context.TODO(),
		pgx.Identifier{tmpTable},
		[]string{
			"job_id",
			"key",
			"value",
		},
		pgx.CopyFromSlice(len(annotations), func(i int) ([]interface{}, error) {
			return []interface{}{
				annotations[i].JobId,
				annotations[i].Key,
				annotations[i].Value,
			}, nil
		}),
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(context.TODO(),
		fmt.Sprintf(`
			INSERT INTO user_annotation_lookup (
				job_id,
				key,
				value
			) SELECT * from %s
			ON CONFLICT DO NOTHING
		`, tmpTable),
	)
	if err != nil {
		return err
	}

	return err
}

func Save(db *pgxpool.Pool, jobs []*LookoutJob, compressor compress.Compressor) error {
	tx, err := db.BeginTx(context.TODO(), pgx.TxOptions{})
	if err != nil {
		panic(err)
	}
	defer func() {
		if err != nil {
			err = tx.Rollback(context.TODO())
		} else {
			err = tx.Commit(context.TODO())
		}
	}()

	err = SaveJobs(tx, jobs, compressor)
	if err != nil {
		return err
	}
	err = SaveJobRuns(tx, jobs)
	if err != nil {
		return err
	}
	err = SaveJobAnnotations(tx, jobs)
	if err != nil {
		return err
	}

	return err
}

func SaveBatched(db *pgxpool.Pool, jobs []*LookoutJob, compressor compress.Compressor, batchSize int) error {
	batches := Batch(jobs, batchSize)
	var ret error = nil
	fmt.Println("Saving to database")
	for i, batch := range batches {
		err := Save(db, batch, compressor)
		if err != nil {
			fmt.Println(err)
			ret = err
		}
		fmt.Printf("%d/%d, batch size: %d\n", i+1, len(batches), batchSize)
	}
	return ret
}

func ExplainAnalyzeQuery(db pgxtype.Querier, query string, args ...interface{}) ([]string, error) {
	rows, err := db.Query(context.TODO(), "EXPLAIN ANALYZE "+query, args...)
	if err != nil {
		return nil, err
	}

	out := []string{}
	for rows.Next() {
		var s string
		if err = rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}

	return out, nil
}

func JobsInQueue() string {
	return `
		SELECT
			j.job_id,
			j.queue,
			j.owner,
			j.jobset,
			j.cpu,
			j.memory,
			j.ephemeral_storage,
			j.gpu,
			j.priority,
			j.submitted,
			j.cancelled,
			j.state,
			j.last_transition_time,
			j.duplicate,
			jr.run_id,
			jr.pod_number,
			jr.cluster,
			jr.node,
			jr.leased,
			jr.started,
			jr.finished,
			jr.job_run_state,
			jr.exit_code,
			jr.active
		FROM (
		    SELECT
		        job_id,
		    	queue,
				owner,
		    	jobset,
		    	cpu,
		    	memory,
		    	ephemeral_storage,
		    	gpu,
		    	priority,
		    	submitted,
		    	cancelled,
		    	state,
		    	last_transition_time,
		    	duplicate
		    FROM job
		    WHERE queue LIKE $1
		    ORDER BY last_transition_time DESC
		    LIMIT 100
		) AS j
		LEFT JOIN job_run AS jr
		ON j.job_id = jr.job_id
	`
}

func JobsInJobset() string {
	return `
		SELECT
			j.job_id,
			j.queue,
			j.owner,
			j.jobset,
			j.cpu,
			j.memory,
			j.ephemeral_storage,
			j.gpu,
			j.priority,
			j.submitted,
			j.cancelled,
			j.state,
			j.last_transition_time,
			j.duplicate,
			jr.run_id,
			jr.pod_number,
			jr.cluster,
			jr.node,
			jr.leased,
			jr.started,
			jr.finished,
			jr.job_run_state,
			jr.exit_code,
			jr.active
		FROM (
		    SELECT
		        job_id,
		    	queue,
				owner,
		    	jobset,
		    	cpu,
		    	memory,
		    	ephemeral_storage,
		    	gpu,
		    	priority,
		    	submitted,
		    	cancelled,
		    	state,
		    	last_transition_time,
		    	duplicate
		    FROM job
		    WHERE jobset LIKE $1
		    ORDER BY last_transition_time DESC, jobset
		    LIMIT 100
		) AS j
		LEFT JOIN job_run AS jr
		ON j.job_id = jr.job_id
	`
}

func JobsOnNode() string {
	return `
		SELECT
			j.job_id,
			j.queue,
			j.owner,
			j.jobset,
			j.cpu,
			j.memory,
			j.ephemeral_storage,
			j.gpu,
			j.priority,
			j.submitted,
			j.cancelled,
			j.state,
			j.last_transition_time,
			j.duplicate,
			jr.run_id,
			jr.pod_number,
			jr.cluster,
			jr.node,
			jr.leased,
			jr.started,
			jr.finished,
			jr.job_run_state,
			jr.exit_code,
			jr.active
		FROM 
		job AS j
		LEFT JOIN job_run AS jr
		ON j.job_id = jr.job_id
		RIGHT JOIN (
		    SELECT DISTINCT ON (job_id)
		        job_id, node
		    FROM job_run
		    WHERE node LIKE $1
		    LIMIT 100
		) AS jr_node_select
		ON j.job_id = jr_node_select.job_id
	`
}

func GroupByQueue() string {
	return `
		SELECT
			queue,
			COUNT(*) AS count
		FROM job
		GROUP BY queue
	`
}

func GroupByState() string {
	return `
		SELECT
			state,
			COUNT(*) AS count
		FROM job
		GROUP BY state
	`
}

func GroupByJobsetAnnotationState() string {
	return `
		SELECT
			j.jobset,
			ual.value,
			j.state,
			COUNT(*) AS count
		FROM job AS j
		INNER JOIN user_annotation_lookup AS ual
		ON j.job_id = ual.job_id
		WHERE ual.key = $1 AND j.queue LIKE $2
		GROUP BY (j.jobset, ual.value, j.state)
	`
}

func PrintStrings(arr []string, indent int) {
	for _, s := range arr {
		fmt.Println(strings.Repeat("\t", indent) + s)
	}
}

func RunQueryTests(db *pgxpool.Pool, queryTests []*QueryTest) {
	for _, test := range queryTests {
		fmt.Println(test.Name)
		result, err := ExplainAnalyzeQuery(db, test.Query, test.Args...)
		if err != nil {
			fmt.Println(err)
			continue
		}
		PrintStrings(result, 1)
	}
}

func SelectString(db pgxtype.Querier, query string) (string, error) {
	rows, err := db.Query(context.TODO(), query)
	if err != nil {
		return "", err
	}
	for rows.Next() {
		var s string
		err := rows.Scan(&s)
		if err != nil {
			return "", err
		}
		return s, nil
	}
	return "", fmt.Errorf("no string found")
}
