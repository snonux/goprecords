package goprecords

import (
	"context"
	"database/sql"

	"codeberg.org/snonux/goprecords/internal/storage"
	_ "modernc.org/sqlite"
)

// OpenDB opens the SQLite database at path, creating the file if needed.
func OpenDB(ctx context.Context, path string) (*sql.DB, error) {
	return storage.Open(ctx, path)
}

// CreateSchema creates the record table and indexes (idempotent).
func CreateSchema(ctx context.Context, db *sql.DB) error {
	return storage.CreateSchema(ctx, db)
}

// ResetRecords removes all rows so import is repeatable.
func ResetRecords(ctx context.Context, db *sql.DB) error {
	return storage.ResetRecords(ctx, db)
}

// ImportFromDir reads all .records files from statsDir and inserts into the DB.
// Resets the record table first so the run is repeatable.
func ImportFromDir(ctx context.Context, db *sql.DB, statsDir string) error {
	return storage.ImportFromDir(ctx, db, statsDir)
}

// LoadAggregates reads all rows from the DB and builds Aggregates (same shape as file-based aggregation).
func LoadAggregates(ctx context.Context, db *sql.DB) (*Aggregates, error) {
	records, err := storage.LoadRecords(ctx, db)
	if err != nil {
		return nil, err
	}
	out := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	hostMaxBoot := make(map[string]int64)
	hostLastKernel := make(map[string]string)

	for _, rec := range records {
		if rec.BootTime >= uint64(hostMaxBoot[rec.Host]) {
			hostMaxBoot[rec.Host] = int64(rec.BootTime)
			hostLastKernel[rec.Host] = rec.OS
		}
		if _, ok := out.Host[rec.Host]; !ok {
			out.Host[rec.Host] = NewHostAggregate(rec.Host, "")
		}
		out.Host[rec.Host].AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.Kernel, rec.OS).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelName, rec.KernelName).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelMajor, rec.KernelMajor).AddRecord(rec.Uptime, rec.BootTime)
	}
	for host, h := range out.Host {
		h.LastKernel = hostLastKernel[host]
	}
	return out, nil
}
