package goprecords

import (
	"context"
	"fmt"
	"os"
)

// RunIntegrationTests runs integration tests against fixture data.
// It returns an error if any test fails.
func RunIntegrationTests(fixturesDir string) error {
	ctx := context.Background()
	aggr := NewAggregator(fixturesDir)
	aggregates, err := aggr.Aggregate(ctx)
	if err != nil {
		return fmt.Errorf("aggregate: %w", err)
	}
	failed := 0
	failed += testReportFixtures(aggregates, fixturesDir)
	failed += testStatsOrder()
	failed += testImportExport(ctx, aggregates, fixturesDir)
	if failed > 0 {
		return fmt.Errorf("%d integration test(s) failed", failed)
	}
	fmt.Println("ok")
	return nil
}

func testReportFixtures(aggregates *Aggregates, fixturesDir string) int {
	limit := uint(3)
	categories := []Category{CategoryHost, CategoryKernel, CategoryKernelMajor, CategoryKernelName}
	metrics := []Metric{MetricBoots, MetricUptime, MetricScore, MetricDowntime, MetricLifespan}
	formats := []OutputFormat{FormatPlaintext, FormatMarkdown, FormatGemtext}
	failed := 0
	for _, cat := range categories {
		for _, met := range metrics {
			if cat != CategoryHost && (met == MetricDowntime || met == MetricLifespan) {
				continue
			}
			for _, outFmt := range formats {
				var report string
				if cat == CategoryHost {
					report = NewHostReporter(aggregates, limit, met, outFmt, 1).Report()
				} else {
					report = NewReporter(aggregates, cat, limit, met, outFmt, 1).Report()
				}
				expectedPath := fmt.Sprintf("%s/%s.%s.%s.expected", fixturesDir, cat, met, outFmt)
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
	return failed
}

func testStatsOrder() int {
	failed := 0
	if _, err := ParseStatsOrder("Host:Uptime,Host:Boots"); err != nil {
		fmt.Printf("FAIL: parse Host:Uptime,Host:Boots: %v\n", err)
		failed++
	}
	merged, _ := StatsOrderList("Host:Uptime")
	if len(merged) == 0 || merged[0].Category != CategoryHost || merged[0].Metric != MetricUptime {
		fmt.Printf("FAIL: stats-order custom first entry\n")
		failed++
	}
	for _, bad := range []string{"Host", "Bad:Uptime", "Kernel:Downtime", "Host:Nope"} {
		if _, err := ParseStatsOrder(bad); err == nil {
			fmt.Printf("FAIL: parse %q should error\n", bad)
			failed++
		}
	}
	return failed
}

func testImportExport(ctx context.Context, aggregates *Aggregates, fixturesDir string) int {
	tmpDB := fixturesDir + "/test_import.db"
	os.Remove(tmpDB)
	failed := 0
	db, err := OpenDB(ctx, tmpDB)
	if err != nil {
		fmt.Printf("FAIL: open tmp db: %v\n", err)
		return 1
	}
	defer func() {
		db.Close()
		os.Remove(tmpDB)
	}()
	CreateSchema(ctx, db)
	if err := ImportFromDir(ctx, db, fixturesDir); err != nil {
		fmt.Printf("FAIL: import: %v\n", err)
		return 1
	}
	aggFromDB, err := LoadAggregates(ctx, db)
	if err != nil {
		fmt.Printf("FAIL: load: %v\n", err)
		return 1
	}
	limit := uint(3)
	reportFromDB := NewHostReporter(aggFromDB, limit, MetricUptime, FormatPlaintext, 1).Report()
	reportFromMem := NewHostReporter(aggregates, limit, MetricUptime, FormatPlaintext, 1).Report()
	if reportFromDB != reportFromMem {
		fmt.Printf("FAIL: import/query report differs from in-memory\n--- from DB:\n%s--- from memory:\n%s\n", reportFromDB, reportFromMem)
		failed++
	}
	return failed
}
