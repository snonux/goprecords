package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/snonux/goprecords/internal/storage"
)

func runExclude(args []string) error {
	fs := flag.NewFlagSet("exclude", flag.ExitOnError)
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	reason := fs.String("reason", "", "Reason for exclusion")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "exclude: hostname required")
		fs.Usage()
		return fmt.Errorf("missing hostname")
	}
	host := fs.Arg(0)
	ctx := context.Background()
	db, err := storage.Open(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := storage.CreateSchema(ctx, db); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := storage.AddExcludedHost(ctx, db, host, *reason); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "excluded host %q from alerts\n", host)
	return nil
}

func runUnexclude(args []string) error {
	fs := flag.NewFlagSet("unexclude", flag.ExitOnError)
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "unexclude: hostname required")
		fs.Usage()
		return fmt.Errorf("missing hostname")
	}
	host := fs.Arg(0)
	ctx := context.Background()
	db, err := storage.Open(ctx, *dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := storage.CreateSchema(ctx, db); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	if err := storage.RemoveExcludedHost(ctx, db, host); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "removed host %q from exclusion list\n", host)
	return nil
}

func runListExcluded(args []string) error {
	fs := flag.NewFlagSet("list-excluded", flag.ExitOnError)
	dbPath := fs.String("db", "goprecords.db", "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return err
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
	hosts, err := storage.LoadExcludedHosts(ctx, db)
	if err != nil {
		return err
	}
	if len(hosts) == 0 {
		fmt.Println("no excluded hosts")
		return nil
	}
	for _, h := range hosts {
		t := time.Unix(h.ExcludedAt, 0).UTC().Format("2006-01-02")
		if h.Reason != "" {
			fmt.Printf("%-40s  excluded=%s  reason=%s\n", h.Host, t, h.Reason)
		} else {
			fmt.Printf("%-40s  excluded=%s\n", h.Host, t)
		}
	}
	return nil
}
