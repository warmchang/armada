package main

import (
	"context"
	"github.com/G-Research/armada/internal/lookout/performance"
)

func main() {
	db := performance.OpenDb()
	_, err := db.Exec(context.TODO(), `
		DROP SCHEMA public CASCADE;
		CREATE SCHEMA public;
		GRANT ALL ON SCHEMA public TO postgres;
		GRANT ALL ON SCHEMA public TO public;
	`)
	performance.PanicOnErr(err)
}
