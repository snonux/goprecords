package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"codeberg.org/snonux/goprecords/internal/goprecords"
	"codeberg.org/snonux/goprecords/internal/storage"
)

func runQuery(args []string) error {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	rf := goprecords.RegisterReportFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	aggregates, err := goprecords.LoadAggregates(ctx, db)
	if err != nil {
		return fmt.Errorf("load: %w", err)
	}
	cfg, err := rf.Parse()
	if err != nil {
		return err
	}
	return goprecords.WriteReports(os.Stdout, aggregates, cfg)
}
