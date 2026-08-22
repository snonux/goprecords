package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/snonux/goprecords/internal/authkeys"
)

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
