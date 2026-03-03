package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/goprecords/internal/goprecords"
	"github.com/goprecords/internal/version"
)

// Execute parses command‑line arguments and runs the appropriate sub‑command.
func Execute(args []string) error {
	// Handle version flag early.
	if len(args) > 0 && (args[0] == "-version" || args[0] == "--version") {
		fmt.Println(version.Version)
		return nil
	}

	// No subcommand – treat args as flags for a direct report from files.
	if len(args) == 0 {
		return runReportFromFiles(nil)
	}

	switch args[0] {
	case "import":
		return runImport(args[1:])
	case "query":
		return runQuery(args[1:])
	case "test":
		return runTests()
	default:
		return runReportFromFiles(args)
	}
}

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
	db, err := goprecords.OpenDB(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := goprecords.CreateSchema(ctx, db); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := goprecords.ImportFromDir(ctx, db, *statsDir); err != nil {
		return fmt.Errorf("import: %w", err)
	}
	fmt.Fprintf(os.Stderr, "imported %s into %s\n", *statsDir, *dbPath)
	return nil
}

func runQuery(args []string) error {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	rf := goprecords.RegisterReportFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx := context.Background()
	db, err := goprecords.OpenDB(ctx, *dbPath)
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

func runReportFromFiles(args []string) error {
	if args == nil {
		args = []string{}
	}
	fs := flag.NewFlagSet("goprecords", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "The uptimed raw record input dir (required)")
	rf := goprecords.RegisterReportFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *statsDir == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -stats-dir")
		fs.Usage()
		return fmt.Errorf("missing -stats-dir")
	}
	cfg, err := rf.Parse()
	if err != nil {
		return err
	}
	ctx := context.Background()
	aggr := goprecords.NewAggregator(*statsDir)
	aggregates, err := aggr.Aggregate(ctx)
	if err != nil {
		return err
	}
	return goprecords.WriteReports(os.Stdout, aggregates, cfg)
}

func runTests() error {
	return goprecords.RunIntegrationTests("./fixtures")
}
