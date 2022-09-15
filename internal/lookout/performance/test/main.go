package main

import (
	"github.com/G-Research/armada/internal/lookout/performance"
)

func main() {
	db := performance.OpenDb()

	queue, err := performance.SelectString(db, `
		SELECT queue FROM job LIMIT 1
	`)
	performance.PanicOnErr(err)
	jobset, err := performance.SelectString(db, `
		SELECT jobset FROM job LIMIT 1
	`)
	performance.PanicOnErr(err)
	node, err := performance.SelectString(db, `
		SELECT node FROM job_run LIMIT 1
	`)
	performance.PanicOnErr(err)
	annotationKey, err := performance.SelectString(db, `
		SELECT key FROM user_annotation_lookup LIMIT 1
	`)
	performance.PanicOnErr(err)

	queryTests := []*performance.QueryTest{
		{
			Name:  "queue exact",
			Query: performance.JobsInQueue(),
			Args:  []interface{}{queue},
		},
		{
			Name:  "queue starts with",
			Query: performance.JobsInQueue(),
			Args:  []interface{}{queue[:len(queue)-1] + "%"},
		},
		{
			Name:  "jobset exact",
			Query: performance.JobsInJobset(),
			Args:  []interface{}{jobset},
		},
		{
			Name:  "jobset starts with",
			Query: performance.JobsInJobset(),
			Args:  []interface{}{jobset[:len(jobset)-1] + "%"},
		},
		{
			Name:  "node exact",
			Query: performance.JobsOnNode(),
			Args:  []interface{}{node},
		},
		{
			Name:  "node starts with",
			Query: performance.JobsOnNode(),
			Args:  []interface{}{node[:len(node)-1] + "%"},
		},
		{
			Name:  "group by queue",
			Query: performance.GroupByQueue(),
			Args:  []interface{}{},
		},
		{
			Name:  "group by state",
			Query: performance.GroupByState(),
			Args:  []interface{}{},
		},
		{
			Name:  "group by jobset annotation state",
			Query: performance.GroupByJobsetAnnotationState(),
			Args:  []interface{}{annotationKey, queue},
		},
	}

	performance.RunQueryTests(db, queryTests)
}
