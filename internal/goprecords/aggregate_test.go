package goprecords

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNewAggregator(t *testing.T) {
	agg := NewAggregator("./test")
	if agg.statsDir != "./test" {
		t.Errorf("expected statsDir ./test, got %q", agg.statsDir)
	}
}

func TestAggregateInvalidDir(t *testing.T) {
	agg := NewAggregator("/nonexistent/path")
	ctx := context.Background()

	_, err := agg.Aggregate(ctx)
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestAggregateFixtures(t *testing.T) {
	fixturesPath := "fixtures"
	if _, err := os.Stat(fixturesPath); err != nil {
		fixturesPath = "../../../fixtures"
	}

	if _, err := os.Stat(fixturesPath); err != nil {
		t.Skipf("skipping test, fixtures directory not found")
	}

	agg := NewAggregator(fixturesPath)
	ctx := context.Background()

	aggregates, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("failed to aggregate fixtures: %v", err)
	}

	if aggregates == nil {
		t.Error("expected non-nil aggregates")
	}

	if len(aggregates.Host) == 0 {
		t.Error("expected hosts in aggregates")
	}

	if len(aggregates.Kernel) == 0 {
		t.Error("expected kernels in aggregates")
	}
}

func TestAggregateFixturesContent(t *testing.T) {
	fixturesPath := "fixtures"
	if _, err := os.Stat(fixturesPath); err != nil {
		fixturesPath = "../../../fixtures"
	}

	if _, err := os.Stat(fixturesPath); err != nil {
		t.Skipf("skipping test, fixtures directory not found")
	}

	agg := NewAggregator(fixturesPath)
	ctx := context.Background()

	aggregates, err := agg.Aggregate(ctx)
	if err != nil {
		t.Fatalf("failed to aggregate fixtures: %v", err)
	}

	// Check a specific host
	if host, ok := aggregates.Host["earth"]; ok {
		if host.Boots == 0 {
			t.Error("expected non-zero boots for earth")
		}
		if host.Uptime == 0 {
			t.Error("expected non-zero uptime for earth")
		}
		if host.LastKernel == "" {
			t.Error("expected non-empty LastKernel for earth")
		}
	} else {
		t.Error("expected earth host in aggregates")
	}
}

func TestGetOrNewAggregate(t *testing.T) {
	m := make(map[string]*Aggregate)

	agg1 := getOrNewAggregate(m, "kernel1")
	if agg1.Name != "kernel1" {
		t.Errorf("expected name kernel1, got %q", agg1.Name)
	}

	agg2 := getOrNewAggregate(m, "kernel1")
	if agg2 != agg1 {
		t.Error("expected same aggregate on second call")
	}

	if len(m) != 1 {
		t.Errorf("expected 1 entry in map, got %d", len(m))
	}
}

func TestLastKernelFromFile(t *testing.T) {
	// Test with a fixture file
	testFile := "fixtures/earth.records"
	if _, err := os.Stat(testFile); err != nil {
		testFile = "../../../fixtures/earth.records"
	}

	if _, err := os.Stat(testFile); err != nil {
		t.Skipf("skipping test, fixture file not found")
	}

	kernel, err := lastKernelFromFile(context.Background(), testFile)
	if err != nil {
		t.Fatalf("failed to get last kernel: %v", err)
	}

	if kernel == "" {
		t.Error("expected non-empty kernel string")
	}
}

func TestLastKernelFromFileNonExistent(t *testing.T) {
	_, err := lastKernelFromFile(context.Background(), "/nonexistent/file.records")
	if err == nil {
		t.Error("expected error for non-existent file")
	}
}

func TestProcessRecordsFile(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.records")

	content := []byte("86400:1000000:Linux 5.10.0-test\n" +
		"86400:1000001:Linux 5.10.0-test\n")

	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}

	// Add host
	aggs.Host["test"] = NewHostAggregate("test", "")

	ctx := context.Background()
	err := processRecordsFile(ctx, testFile, "test", aggs)

	if err != nil {
		t.Fatalf("failed to process records: %v", err)
	}

	if aggs.Host["test"].Boots != 2 {
		t.Errorf("expected 2 boots, got %d", aggs.Host["test"].Boots)
	}
}

func TestContextCancellation(t *testing.T) {
	agg := NewAggregator("./fixtures")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := agg.Aggregate(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}
