package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/snonux/goprecords/internal/authkeys"
	"codeberg.org/snonux/goprecords/internal/daemon"
	"codeberg.org/snonux/goprecords/internal/goprecords"
	"codeberg.org/snonux/goprecords/internal/storage"
	"codeberg.org/snonux/goprecords/internal/version"
)

// Execute parses command‑line arguments and runs the appropriate sub‑command.
func Execute(args []string) error {
	// Handle version flag early.
	if len(args) > 0 && (args[0] == "-version" || args[0] == "--version") {
		fmt.Println(version.Version)
		return nil
	}
	if len(args) >= 1 && (args[0] == "--create-client-key" || args[0] == "-create-client-key") {
		if len(args) < 2 {
			return fmt.Errorf("create-client-key: hostname required")
		}
		return runCreateClientKey(args[1], args[2:])
	}
	if len(args) > 0 && (args[0] == "-daemon" || args[0] == "--daemon") {
		return runDaemon(args[1:])
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

func defaultListenFromEnv() string {
	if s := os.Getenv("GOPRECORDS_LISTEN"); s != "" {
		return s
	}
	return ":8080"
}

func runDaemon(args []string) error {
	fs := flag.NewFlagSet("daemon", flag.ExitOnError)
	fs.SetOutput(os.Stdout)
	statsDir := fs.String("stats-dir", os.Getenv("GOPRECORDS_STATS_DIR"), "Uptimed stats directory (required; env GOPRECORDS_STATS_DIR)")
	listen := fs.String("listen", defaultListenFromEnv(), "TCP listen address (env GOPRECORDS_LISTEN, default :8080)")
	authDB := fs.String("auth-db", "", "SQLite file for upload API keys (default: <stats-dir>/goprecords-auth.db)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *statsDir == "" {
		fmt.Fprintln(os.Stdout, "daemon: missing required flag: -stats-dir (or GOPRECORDS_STATS_DIR)")
		fs.Usage()
		return fmt.Errorf("missing -stats-dir")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	err := daemon.Run(ctx, daemon.Config{StatsDir: *statsDir, Addr: *listen, AuthDB: *authDB})
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func runCreateClientKey(hostname string, args []string) error {
	if hostname == "" {
		return fmt.Errorf("create-client-key: hostname required")
	}
	fs := flag.NewFlagSet("create-client-key", flag.ExitOnError)
	fs.SetOutput(os.Stderr)
	statsDir := fs.String("stats-dir", "", "Uptimed stats directory (sets default auth-db path)")
	authDB := fs.String("auth-db", "", "SQLite file for upload API keys (default: <stats-dir>/goprecords-auth.db)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	authPath := *authDB
	if authPath == "" {
		if *statsDir == "" {
			fmt.Fprintln(os.Stderr, "create-client-key: need -stats-dir or -auth-db")
			fs.Usage()
			return fmt.Errorf("missing -stats-dir or -auth-db")
		}
		authPath = authkeys.DefaultPath(*statsDir)
	}
	ctx := context.Background()
	store, err := authkeys.OpenStore(ctx, authPath)
	if err != nil {
		return fmt.Errorf("open auth db: %w", err)
	}
	defer store.Close()
	if err := store.EnsureSchema(ctx); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	token, err := store.CreateKey(ctx, hostname)
	if err != nil {
		return fmt.Errorf("create key: %w", err)
	}
	fmt.Println(token)
	return nil
}
