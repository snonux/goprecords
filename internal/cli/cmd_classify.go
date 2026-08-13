package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"sort"

	"codeberg.org/snonux/goprecords/internal/hostclass"
	"codeberg.org/snonux/goprecords/internal/storage"
)

func runClassify(args []string) error {
	fs := flag.NewFlagSet("classify", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "Stats directory holding the HOST.class files")
	dbPath := fs.String("db", "", "SQLite database to update as well (optional)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "classify: hostname and class required (server, workstation, hybrid or unknown)")
		fs.Usage()
		return fmt.Errorf("missing hostname or class")
	}
	if *statsDir == "" && *dbPath == "" {
		fmt.Fprintln(os.Stderr, "classify: -stats-dir or -db required")
		return fmt.Errorf("missing -stats-dir")
	}
	host := fs.Arg(0)
	class, err := hostclass.Parse(fs.Arg(1))
	if err != nil {
		return err
	}
	if *statsDir != "" {
		if err := hostclass.WriteFile(*statsDir, host, class); err != nil {
			return err
		}
	}
	if *dbPath != "" {
		if err := storeHostClass(*dbPath, host, class); err != nil {
			return err
		}
	}
	fmt.Fprintf(os.Stderr, "classified host %q as %s\n", host, class)
	return nil
}

func storeHostClass(dbPath, host string, class hostclass.Class) error {
	ctx := context.Background()
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	if err := storage.CreateSchema(ctx, db); err != nil {
		return fmt.Errorf("schema: %w", err)
	}
	return storage.SetHostClass(ctx, db, host, class.String())
}

func runListClasses(args []string) error {
	fs := flag.NewFlagSet("list-classes", flag.ExitOnError)
	statsDir := fs.String("stats-dir", "", "Stats directory holding the HOST.class files")
	dbPath := fs.String("db", "", "SQLite database to read instead of a stats directory")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *statsDir == "" && *dbPath == "" {
		fmt.Fprintln(os.Stderr, "list-classes: -stats-dir or -db required")
		return fmt.Errorf("missing -stats-dir")
	}
	classes, err := loadHostClasses(*statsDir, *dbPath)
	if err != nil {
		return err
	}
	if len(classes) == 0 {
		fmt.Println("no classified hosts")
		return nil
	}
	hosts := make([]string, 0, len(classes))
	for host := range classes {
		hosts = append(hosts, host)
	}
	sort.Strings(hosts)
	for _, host := range hosts {
		fmt.Printf("%-40s  %s\n", host, classes[host])
	}
	return nil
}

// loadHostClasses reads classifications from the stats directory when given,
// otherwise from the database.
func loadHostClasses(statsDir, dbPath string) (map[string]string, error) {
	if statsDir != "" {
		classes, err := hostclass.Load(statsDir)
		if err != nil {
			return nil, err
		}
		out := make(map[string]string, len(classes))
		for host, c := range classes {
			out[host] = c.String()
		}
		return out, nil
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	defer db.Close()
	return storage.LoadHostClasses(ctx, db)
}
