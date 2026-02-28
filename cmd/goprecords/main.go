// Program goprecords generates uptime reports from uptimed record files or a SQLite database.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/goprecords/internal/goprecords"
	"github.com/goprecords/internal/version"
)

const defaultDB = "goprecords.db"

func main() {
	if len(os.Args) > 1 && (os.Args[1] == "-version" || os.Args[1] == "--version") {
		fmt.Println(version.Version)
		return
	}

	if len(os.Args) < 2 {
		runReportFromFiles(nil)
		return
	}

	switch os.Args[1] {
	case "import":
		runImport(os.Args[2:])
	case "query":
		runQuery(os.Args[2:])
	case "test":
		runTests()
	default:
		runReportFromFiles(os.Args[1:])
	}
}

func runImport(args []string) {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "Directory containing .records files (required)")
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	fs.Parse(args)

	if *statsDir == "" {
		fmt.Fprintln(os.Stderr, "import: missing required flag: -stats-dir")
		fs.Usage()
		os.Exit(1)
	}
	db, err := goprecords.OpenDB(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close()
	ctx := context.Background()
	if err := goprecords.CreateSchema(ctx, db); err != nil {
		fmt.Fprintln(os.Stderr, "schema:", err)
		os.Exit(1)
	}
	if err := goprecords.ImportFromDir(ctx, db, *statsDir); err != nil {
		fmt.Fprintln(os.Stderr, "import:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "imported %s into %s\n", *statsDir, *dbPath)
}

func runQuery(args []string) {
	fs := flag.NewFlagSet("query", flag.ExitOnError)
	dbPath := fs.String("db", defaultDB, "SQLite database path")
	rf := goprecords.RegisterReportFlags(fs)
	fs.Parse(args)

	db, err := goprecords.OpenDB(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open db:", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()
	aggregates, err := goprecords.LoadAggregates(ctx, db)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load:", err)
		os.Exit(1)
	}

	cfg, err := rf.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := goprecords.WriteReports(os.Stdout, aggregates, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runReportFromFiles(args []string) {
	if args == nil {
		args = []string{}
	}
	fs := flag.NewFlagSet("goprecords", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "The uptimed raw record input dir (required)")
	rf := goprecords.RegisterReportFlags(fs)
	fs.Parse(args)

	if *statsDir == "" {
		fmt.Fprintln(os.Stderr, "missing required flag: -stats-dir")
		fs.Usage()
		os.Exit(1)
	}

	cfg, err := rf.Parse()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()
	aggr := goprecords.NewAggregator(*statsDir)
	aggregates, err := aggr.Aggregate(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := goprecords.WriteReports(os.Stdout, aggregates, cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runTests() {
	if err := goprecords.RunIntegrationTests("./fixtures"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
