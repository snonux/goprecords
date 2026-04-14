package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
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
