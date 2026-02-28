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
	ctx := context.Background()
	aggr := goprecords.NewAggregator("./fixtures")
	aggregates, err := aggr.Aggregate(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	limit := uint(3)
	categories := []goprecords.Category{goprecords.CategoryHost, goprecords.CategoryKernel, goprecords.CategoryKernelMajor, goprecords.CategoryKernelName}
	metrics := []goprecords.Metric{goprecords.MetricBoots, goprecords.MetricUptime, goprecords.MetricScore, goprecords.MetricDowntime, goprecords.MetricLifespan}
	formats := []goprecords.OutputFormat{goprecords.FormatPlaintext, goprecords.FormatMarkdown, goprecords.FormatGemtext}
	failed := 0
	for _, cat := range categories {
		for _, met := range metrics {
			if cat != goprecords.CategoryHost && (met == goprecords.MetricDowntime || met == goprecords.MetricLifespan) {
				continue
			}
			for _, outFmt := range formats {
				var report string
				if cat == goprecords.CategoryHost {
					report = goprecords.NewHostReporter(aggregates, limit, met, outFmt, 1).Report()
				} else {
					report = goprecords.NewReporter(aggregates, cat, limit, met, outFmt, 1).Report()
				}
				expectedPath := fmt.Sprintf("./fixtures/%s.%s.%s.expected", cat, met, outFmt)
				expected, err := os.ReadFile(expectedPath)
				if err != nil {
					fmt.Printf("FAIL: read %s: %v\n", expectedPath, err)
					failed++
					continue
				}
				if report != string(expected) {
					fmt.Printf("FAIL: %s\n--- got:\n%s--- expected:\n%s\n", expectedPath, report, string(expected))
					failed++
				}
			}
		}
	}
	if _, err := goprecords.ParseStatsOrder("Host:Uptime,Host:Boots"); err != nil {
		fmt.Printf("FAIL: parse Host:Uptime,Host:Boots: %v\n", err)
		failed++
	}
	merged, _ := goprecords.StatsOrderList("Host:Uptime")
	if len(merged) == 0 || merged[0].Category != goprecords.CategoryHost || merged[0].Metric != goprecords.MetricUptime {
		fmt.Printf("FAIL: stats-order custom first entry\n")
		failed++
	}
	for _, bad := range []string{"Host", "Bad:Uptime", "Kernel:Downtime", "Host:Nope"} {
		if _, err := goprecords.ParseStatsOrder(bad); err == nil {
			fmt.Printf("FAIL: parse %q should error\n", bad)
			failed++
		}
	}
	tmpDB := "./fixtures/test_import.db"
	os.Remove(tmpDB)
	db, err := goprecords.OpenDB(tmpDB)
	if err != nil {
		fmt.Printf("FAIL: open tmp db: %v\n", err)
		failed++
	} else {
		goprecords.CreateSchema(ctx, db)
		if err := goprecords.ImportFromDir(ctx, db, "./fixtures"); err != nil {
			fmt.Printf("FAIL: import: %v\n", err)
			failed++
		} else {
			aggFromDB, err := goprecords.LoadAggregates(ctx, db)
			if err != nil {
				fmt.Printf("FAIL: load: %v\n", err)
				failed++
			} else {
				reportFromDB := goprecords.NewHostReporter(aggFromDB, limit, goprecords.MetricUptime, goprecords.FormatPlaintext, 1).Report()
				reportFromMem := goprecords.NewHostReporter(aggregates, limit, goprecords.MetricUptime, goprecords.FormatPlaintext, 1).Report()
				if reportFromDB != reportFromMem {
					fmt.Printf("FAIL: import/query report differs from in-memory\n--- from DB:\n%s--- from memory:\n%s\n", reportFromDB, reportFromMem)
					failed++
				}
			}
		}
		db.Close()
		os.Remove(tmpDB)
	}
	if failed > 0 {
		os.Exit(1)
	}
	fmt.Println("ok")
}
