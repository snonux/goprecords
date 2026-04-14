package goprecords

import (
	"context"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/goprecords/internal/storage"
)

func TestLoadAggregates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := storage.CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 1000, 2000, "Linux 5.10", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 2000, 3000, "Linux 5.11", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	aggs, err := LoadAggregates(ctx, db)
	if err != nil {
		t.Fatalf("failed to load aggregates: %v", err)
	}

	if aggs == nil {
		t.Error("expected non-nil aggregates")
	}

	if len(aggs.Host) != 1 {
		t.Errorf("expected 1 host, got %d", len(aggs.Host))
	}

	if host, ok := aggs.Host["host1"]; ok {
		if host.Boots != 2 {
			t.Errorf("expected 2 boots, got %d", host.Boots)
		}
		if host.Uptime != 3000 {
			t.Errorf("expected uptime 3000, got %d", host.Uptime)
		}
		if host.LastKernel != "Linux 5.11" {
			t.Errorf("LastKernel = %q, want %q (latest boot_time row)", host.LastKernel, "Linux 5.11")
		}
	}
}

func TestLoadAggregatesLastKernelMaxBootNearInt64(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := storage.CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	early := math.MaxInt64 - 1
	late := math.MaxInt64

	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 100, early, "Linux early", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 100, late, "Linux late", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	aggs, err := LoadAggregates(ctx, db)
	if err != nil {
		t.Fatalf("failed to load aggregates: %v", err)
	}

	host := aggs.Host["host1"]
	if host == nil {
		t.Fatal("expected host1 aggregate")
	}
	if host.LastKernel != "Linux late" {
		t.Errorf("LastKernel = %q, want %q", host.LastKernel, "Linux late")
	}
}

func TestLoadAggregatesEmptyDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := storage.CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	aggs, err := LoadAggregates(ctx, db)
	if err != nil {
		t.Fatalf("failed to load aggregates: %v", err)
	}

	if aggs == nil {
		t.Error("expected non-nil aggregates")
	}

	if len(aggs.Host) != 0 {
		t.Errorf("expected 0 hosts, got %d", len(aggs.Host))
	}
}

func TestLoadAggregatesWrapsLoadRecordsError(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	ctx := context.Background()
	if err := storage.CreateSchema(ctx, db); err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}
	db.Close()

	_, err = LoadAggregates(ctx, db)
	if err == nil {
		t.Fatal("expected error after db closed")
	}
	if !strings.Contains(err.Error(), "load records:") {
		t.Fatalf("expected load records context in error, got %v", err)
	}
}
