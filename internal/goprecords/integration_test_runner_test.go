package goprecords

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/snonux/goprecords/internal/storage"
)

func TestTestImportExportOnDB_createSchemaError(t *testing.T) {
	ctx := context.Background()
	fixturesDir := filepath.Join("..", "..", "fixtures")
	aggr := NewAggregator(fixturesDir)
	aggregates, err := aggr.Aggregate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "import.db")
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	ctxCanceled, cancel := context.WithCancel(ctx)
	cancel()
	if n := testImportExportOnDB(ctxCanceled, db, dbPath, aggregates, fixturesDir); n != 1 {
		t.Fatalf("testImportExportOnDB: got %d, want 1", n)
	}
}
