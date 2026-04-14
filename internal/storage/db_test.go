package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestOpen_ContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	path := filepath.Join(t.TempDir(), "records.db")
	_, err := Open(ctx, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestOpen_PingFailsOnDirectoryPath(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "isdir")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Open(ctx, dir)
	if err == nil {
		t.Fatal("expected error opening sqlite at directory path")
	}
}

func TestOpen_createsDatabaseFile(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(context.Background(), dbPath)
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

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	err = CreateSchema(ctx, db)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	_, err = db.ExecContext(ctx, "SELECT 1 FROM record LIMIT 1")
	if err != nil {
		t.Fatalf("failed to query record table: %v", err)
	}
}

func TestResetRecords(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	_, err = db.ExecContext(ctx,
		"INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)",
		"host1", 1000, 2000, "Linux 5.10", "Linux", "Linux 5...")
	if err != nil {
		t.Fatalf("failed to insert record: %v", err)
	}

	err = ResetRecords(ctx, db)
	if err != nil {
		t.Fatalf("failed to reset records: %v", err)
	}

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
	tmpDir := t.TempDir()

	recordsFile := filepath.Join(tmpDir, "testhost.records")
	content := []byte("86400:1000000:Linux 5.10.0-test\n" +
		"86400:1000001:Linux 5.10.0-test\n" +
		"86400:1000002:Linux 5.10.0-test\n")

	if err := os.WriteFile(recordsFile, content, 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	err = ImportFromDir(ctx, db, tmpDir)
	if err != nil {
		t.Fatalf("failed to import records: %v", err)
	}

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
	db, err := Open(context.Background(), dbPath)
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

func TestImportFromDir_invalidPath(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	err = ImportFromDir(ctx, db, "/nonexistent/path")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestCreateSchema_closedDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	err = CreateSchema(context.Background(), db)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCreateSchema_contextCanceled(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = CreateSchema(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestCreateSchema_idempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatalf("second CreateSchema: %v", err)
	}
}

func TestResetRecords_noSchema(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	err = ResetRecords(context.Background(), db)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResetRecords_closedDB(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	err = ResetRecords(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRecords_empty(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	recs, err := LoadRecords(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 0 {
		t.Fatalf("len=%d, want 0", len(recs))
	}
}

func TestLoadRecords_orderAndFields(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	ins := `INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)`
	want := []Record{
		{Host: "a", Uptime: 1, BootTime: 100, OS: "Linux 5", KernelName: "Linux", KernelMajor: "Linux 5..."},
		{Host: "a", Uptime: 2, BootTime: 200, OS: "Linux 5", KernelName: "Linux", KernelMajor: "Linux 5..."},
		{Host: "b", Uptime: 3, BootTime: 50, OS: "Linux 5", KernelName: "Linux", KernelMajor: "Linux 5..."},
	}
	if _, err := db.ExecContext(ctx, ins, "b", 3, 50, "Linux 5", "Linux", "Linux 5..."); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, ins, "a", 2, 200, "Linux 5", "Linux", "Linux 5..."); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, ins, "a", 1, 100, "Linux 5", "Linux", "Linux 5..."); err != nil {
		t.Fatal(err)
	}
	got, err := LoadRecords(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(want) {
		t.Fatalf("len(got)=%d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("record %d: %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestLoadRecords_noTable(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = LoadRecords(context.Background(), db)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRecords_contextCanceledBeforeQuery(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = LoadRecords(ctx, db)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestImportFromDir_skipsInvalidLines(t *testing.T) {
	tmpDir := t.TempDir()
	recordsFile := filepath.Join(tmpDir, "h.records")
	content := []byte("not-valid\n86400:1000000:Linux 5.10.0-test\n:::bad\n")
	if err := os.WriteFile(recordsFile, content, 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := ImportFromDir(ctx, db, tmpDir); err != nil {
		t.Fatal(err)
	}
	recs, err := LoadRecords(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("len=%d, want 1", len(recs))
	}
	if recs[0].Host != "h" {
		t.Fatalf("host=%q", recs[0].Host)
	}
}

func TestImportFromDir_contextCanceledBeforeImport(t *testing.T) {
	tmpDir := t.TempDir()
	recordsFile := filepath.Join(tmpDir, "h.records")
	if err := os.WriteFile(recordsFile, []byte("86400:1:Linux 5\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = ImportFromDir(ctx, db, tmpDir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestImportFromDir_pathIsFileNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "notadir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	err = ImportFromDir(context.Background(), db, filePath)
	if err == nil {
		t.Fatal("expected error")
	}
}
