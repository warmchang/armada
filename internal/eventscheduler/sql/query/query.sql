-- -- name: GetRecord :one
-- SELECT * FROM records WHERE id = $1 LIMIT 1;

-- name: ListRuns :many
SELECT * FROM runs ORDER BY run_id;

-- name: SelectNewRunsForExecutor :many
SELECT * FROM runs WHERE (executor = $1 AND sent_to_executor = false);

-- name: SelectRunsFromExecutorAndJobs :many
SELECT * FROM runs WHERE (executor = $1 AND job_id = ANY(sqlc.arg(job_ids)::UUID[]));

-- name: SelectNewRunsForExecutorWithLimit :many
SELECT * FROM runs WHERE (executor = $1 AND sent_to_executor = false) LIMIT $2;

-- name: MarkRunAsSent :exec
UPDATE runs SET sent_to_executor = true WHERE run_id = $1;

-- name: MarkRunsAsSent :exec
UPDATE runs SET sent_to_executor = true WHERE run_id = ANY(sqlc.arg(run_ids)::UUID[]);

-- name: SelectJobsFromIds :many
SELECT * FROM jobs WHERE job_id = ANY(sqlc.arg(job_ids)::UUID[]);
-- SELECT (user_id, groups, queue, submit_message) FROM jobs WHERE job_id = ANY(sqlc.arg(job_ids)::UUID[]);

-- name: SelectQueueJobSetFromId :one
SELECT job_id, queue, job_set FROM jobs where job_id = $1;

-- name: SelectQueueJobSetFromIds :many
SELECT job_id, queue, job_set FROM jobs where job_id = ANY(sqlc.arg(job_ids)::UUID[]);

-- name: ListNodeInfo :many
SELECT * FROM nodeinfo ORDER BY serial;

-- name: SelectNewNodeInfo :many
SELECT * FROM nodeinfo WHERE serial > $1 ORDER BY serial;

-- name: SelectNewJobs :many
SELECT * FROM jobs WHERE serial > $1 ORDER BY serial;

-- name: SelectNewRuns :many
SELECT * FROM runs WHERE serial > $1 ORDER BY serial;

-- name: UpdateJobPriorityById :exec
UPDATE jobs SET priority = $1 WHERE job_id = $2;

-- name: UpdateJobPriorityByJobSet :exec
UPDATE jobs SET priority = $1 WHERE job_set = $2;

-- name: CancelJobById :exec
UPDATE jobs SET cancelled = true WHERE job_id = $1;

-- name: CancelJobsById :exec
UPDATE jobs SET cancelled = true WHERE job_id = ANY(sqlc.arg(job_ids)::UUID[]);

-- name: CancelJobsBySet :exec
UPDATE jobs SET cancelled = true WHERE job_set = $1;

-- name: CancelJobsBySets :exec
UPDATE jobs SET cancelled = true WHERE job_set = ANY(sqlc.arg(job_sets)::text[]);

-- name: MarkJobRunRunningById :exec
UPDATE runs SET running = true WHERE job_id = $1;

-- name: MarkJobRunsRunningById :exec
UPDATE runs SET running = true WHERE job_id = ANY(sqlc.arg(job_ids)::UUID[]);

-- name: MarkJobRunFailedById :exec
UPDATE runs SET error = $1 WHERE job_id = $2;

-- -- name: UpsertRecord :exec
-- INSERT INTO records (id, value, payload) VALUES ($1, $2, $3)
-- ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, payload = EXCLUDED.payload;

-- -- name: UpsertRecords :exec
-- INSERT INTO records (id, value, payload)
-- SELECT unnest(@ids) AS id,
--        unnest(@values) AS names,
--        unnest(@payloads) AS payloads
-- ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value, payload = EXCLUDED.payload;

-- -- name: UpdateRecord :exec
-- UPDATE records SET value = $2, payload = $3 WHERE id = $1;

-- -- name: DeleteRecord :exec
-- DELETE FROM records WHERE id = $1;

-- name: GetTopicMessageIds :many
SELECT * FROM pulsar WHERE topic = $1;

-- name: UpsertMessageId :exec
INSERT INTO pulsar (topic, ledger_id, entry_id, batch_idx, partition_idx) VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (topic, partition_idx) DO UPDATE SET ledger_id = EXCLUDED.ledger_id, entry_id = EXCLUDED.entry_id, batch_idx = EXCLUDED.batch_idx;
