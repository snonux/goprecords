package goprecords

import (
	"context"
	"os"
	"path/filepath"
	"testing"
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
