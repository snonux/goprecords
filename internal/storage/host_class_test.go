package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func newTestDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(context.Background(), filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if err := CreateSchema(context.Background(), db); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db, dir
}

func TestSetAndLoadHostClasses(t *testing.T) {
	ctx := context.Background()
	db, _ := newTestDB(t)
	if err := SetHostClass(ctx, db, "earth", "server"); err != nil {
		t.Fatal(err)
	}
	if err := SetHostClass(ctx, db, "earth", "hybrid"); err != nil {
		t.Fatal(err)
	}
	if err := SetHostClass(ctx, db, "t450", "workstation"); err != nil {
		t.Fatal(err)
	}
	classes, err := LoadHostClasses(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if classes["earth"] != "hybrid" || classes["t450"] != "workstation" || len(classes) != 2 {
		t.Fatalf("got %#v", classes)
	}
}

func TestLoadHostClassesWithoutTable(t *testing.T) {
	ctx := context.Background()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "empty.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	classes, err := LoadHostClasses(ctx, db)
	if err != nil {
		t.Fatalf("expected no error for a database without host_class, got %v", err)
	}
	if len(classes) != 0 {
		t.Fatalf("got %#v, want empty", classes)
	}
}

func TestImportFromDirStoresHostClasses(t *testing.T) {
	ctx := context.Background()
	db, dir := newTestDB(t)
	writeFile(t, filepath.Join(dir, "earth.records"), "86400:1000000:Linux 5.10.0-test\n")
	writeFile(t, filepath.Join(dir, "earth.class"), "server\n")
	writeFile(t, filepath.Join(dir, "broken.class"), "toaster\n")
	// A class set only in the database must survive a re-import.
	if err := SetHostClass(ctx, db, "dbonly", "hybrid"); err != nil {
		t.Fatal(err)
	}
	if err := ImportFromDir(ctx, db, dir); err != nil {
		t.Fatal(err)
	}
	classes, err := LoadHostClasses(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if classes["earth"] != "server" {
		t.Errorf("class of earth = %q, want server", classes["earth"])
	}
	if classes["dbonly"] != "hybrid" {
		t.Errorf("class of dbonly = %q, want hybrid", classes["dbonly"])
	}
	if _, ok := classes["broken"]; ok {
		t.Errorf("invalid class file should be ignored, got %#v", classes)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
