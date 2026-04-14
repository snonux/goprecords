package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"codeberg.org/snonux/goprecords/internal/daemon"
)

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
