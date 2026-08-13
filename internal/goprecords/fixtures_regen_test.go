package goprecords

import (
	"context"
	"fmt"
	"os"
	"testing"
)

// TestRegenFixtures rewrites the .expected report fixtures compared by
// RunIntegrationTests. It is skipped by default and only meant to be run
// (REGEN_FIXTURES=1 go test ./internal/goprecords/ -run TestRegenFixtures)
// after an intended change to the report layout.
func TestRegenFixtures(t *testing.T) {
	if os.Getenv("REGEN_FIXTURES") == "" {
		t.Skip("set REGEN_FIXTURES=1 to regenerate")
	}
	dir := "../../fixtures"
	aggr := NewAggregator(dir)
	aggregates, err := aggr.Aggregate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	limit := uint(3)
	categories := []Category{CategoryHost, CategoryKernel, CategoryKernelMajor, CategoryKernelName}
	metrics := []Metric{MetricBoots, MetricUptime, MetricScore, MetricDowntime, MetricLifespan}
	formats := []OutputFormat{FormatPlaintext, FormatMarkdown, FormatGemtext, FormatHTML}
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
				path := fmt.Sprintf("%s/%s.%s.%s.expected", dir, cat, met, outFmt)
				if err := os.WriteFile(path, []byte(report), 0o644); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
