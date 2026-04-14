package goprecords

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestOpenDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	if db == nil {
		t.Error("expected non-nil database")
	}
}

func TestCreateSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	err = CreateSchema(ctx, db)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Verify schema was created by checking if we can query it
	_, err = db.ExecContext(ctx, "SELECT 1 FROM record LIMIT 1")
	if err != nil {
		t.Fatalf("failed to query record table: %v", err)
	}
}

func TestResetRecords(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert a record
	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 1000, 2000, "Linux 5.10", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert record: %v", err)
	}

	// Reset records
	err = ResetRecords(ctx, db)
	if err != nil {
		t.Fatalf("failed to reset records: %v", err)
	}

	// Verify records are empty
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM record").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count records: %v", err)
	}

	if count != 0 {
		t.Errorf("expected 0 records after reset, got %d", count)
	}
}

func TestImportFromDir(t *testing.T) {
	// Create temp directory with test records
	tmpDir := t.TempDir()

	// Create a test records file
	recordsFile := filepath.Join(tmpDir, "testhost.records")
	content := []byte("86400:1000000:Linux 5.10.0-test\n" +
		"86400:1000001:Linux 5.10.0-test\n" +
		"86400:1000002:Linux 5.10.0-test\n")

	if err := os.WriteFile(recordsFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create database
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Import records
	err = ImportFromDir(ctx, db, tmpDir)
	if err != nil {
		t.Fatalf("failed to import records: %v", err)
	}

	// Verify records were imported
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM record").Scan(&count)
	if err != nil {
		t.Fatalf("failed to count records: %v", err)
	}

	if count != 3 {
		t.Errorf("expected 3 records after import, got %d", count)
	}
}

func TestImportFromFS_MapFS(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("open DB: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("schema: %v", err)
	}
	m := fstest.MapFS{
		"testhost.records": &fstest.MapFile{
			Data: []byte("86400:1000000:Linux 5.10.0-test\n" +
				"86400:1000001:Linux 5.10.0-test\n" +
				"86400:1000002:Linux 5.10.0-test\n"),
			Mode: 0o644,
		},
	}
	if err := ImportFromFS(ctx, db, m); err != nil {
		t.Fatalf("ImportFromFS: %v", err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM record").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
}

func TestImportFromDirInvalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Try to import from non-existent directory
	err = ImportFromDir(ctx, db, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestLoadAggregates(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert some records
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

	// Load aggregates
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

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
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

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Load from empty database
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

	db, err := OpenDB(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
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
