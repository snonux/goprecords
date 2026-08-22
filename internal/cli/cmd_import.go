package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/snonux/goprecords/internal/storage"
)

func runImport(args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "Directory containing .records files (required)")
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *statsDir == "" {
		fmt.Fprintln(os.Stderr, "import: missing required flag: -stats-dir")
		fs.Usage()
		return fmt.Errorf("missing -stats-dir")
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := storage.CreateSchema(ctx, db); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := storage.ImportFromDir(ctx, db, *statsDir); err != nil {
		return fmt.Errorf("import: %w", err)
	}
	fmt.Fprintf(os.Stderr, "imported %s into %s\n", *statsDir, *dbPath)
	return nil
}
