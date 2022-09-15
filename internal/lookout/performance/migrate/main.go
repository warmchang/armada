package main

import (
	"github.com/G-Research/armada/internal/lookout/performance"
	"path/filepath"
)

func main() {
	db := performance.OpenDb()
	migrations := []string{
		filepath.Join("internal", "lookout", "performance", "sql", "simple.sql"),
	}
	performance.MigrateDatabase(db, migrations)
}
