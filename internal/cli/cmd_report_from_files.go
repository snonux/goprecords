package cli

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/snonux/goprecords/internal/goprecords"
)

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
