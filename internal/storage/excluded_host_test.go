package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestAddExcludedHost(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := AddExcludedHost(ctx, db, "host1", "decommissioned"); err != nil {
		t.Fatal(err)
	}
	hosts, err := LoadExcludedHosts(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 {
		t.Fatalf("len=%d, want 1", len(hosts))
	}
	if hosts[0].Host != "host1" || hosts[0].Reason != "decommissioned" {
		t.Fatalf("unexpected host entry: %+v", hosts[0])
	}
}

func TestAddExcludedHost_idempotent(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := AddExcludedHost(ctx, db, "host1", "first"); err != nil {
		t.Fatal(err)
	}
	if err := AddExcludedHost(ctx, db, "host1", "second"); err != nil {
		t.Fatal(err)
	}
	hosts, err := LoadExcludedHosts(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 1 {
		t.Fatalf("len=%d, want 1", len(hosts))
	}
	if hosts[0].Reason != "second" {
		t.Fatalf("reason=%q, want second", hosts[0].Reason)
	}
}

func TestRemoveExcludedHost(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := AddExcludedHost(ctx, db, "host1", ""); err != nil {
		t.Fatal(err)
	}
	if err := RemoveExcludedHost(ctx, db, "host1"); err != nil {
		t.Fatal(err)
	}
	hosts, err := LoadExcludedHosts(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Fatalf("len=%d, want 0", len(hosts))
	}
}

func TestRemoveExcludedHost_nonexistent(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := RemoveExcludedHost(ctx, db, "ghost"); err != nil {
		t.Fatal(err)
	}
}

func TestIsExcludedHost(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := AddExcludedHost(ctx, db, "host1", ""); err != nil {
		t.Fatal(err)
	}
	yes, err := IsExcludedHost(ctx, db, "host1")
	if err != nil {
		t.Fatal(err)
	}
	if !yes {
		t.Fatal("expected host1 to be excluded")
	}
	no, err := IsExcludedHost(ctx, db, "other")
	if err != nil {
		t.Fatal(err)
	}
	if no {
		t.Fatal("expected other to not be excluded")
	}
}

func TestLoadExcludedHosts_empty(t *testing.T) {
	db, err := Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	if err := CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	hosts, err := LoadExcludedHosts(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Fatalf("len=%d, want 0", len(hosts))
	}
}
