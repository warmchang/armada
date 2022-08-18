// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.13.0

package eventscheduler

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgtype"
)

type Executor struct {
	ID             string      `db:"id"`
	Totalresources pgtype.JSON `db:"totalresources"`
	Maxresources   pgtype.JSON `db:"maxresources"`
}

type Job struct {
	Jobid        uuid.UUID `db:"jobid"`
	Jobset       string    `db:"jobset"`
	Queue        string    `db:"queue"`
	Priority     int64     `db:"priority"`
	Message      []byte    `db:"message"`
	Messageindex int64     `db:"messageindex"`
}

type Pulsar struct {
	Topic        string `db:"topic"`
	Ledgerid     int64  `db:"ledgerid"`
	Entryid      int64  `db:"entryid"`
	Batchidx     int32  `db:"batchidx"`
	Partitionidx int32  `db:"partitionidx"`
}

type Queue struct {
	Name   string  `db:"name"`
	Weight float64 `db:"weight"`
}

type Run struct {
	RunID        uuid.UUID   `db:"run_id"`
	JobID        uuid.UUID   `db:"job_id"`
	Executor     string      `db:"executor"`
	Assignment   pgtype.JSON `db:"assignment"`
	Deleted      bool        `db:"deleted"`
	LastModified time.Time   `db:"last_modified"`
}