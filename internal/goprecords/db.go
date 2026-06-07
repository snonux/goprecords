package goprecords

import (
	"context"
	"database/sql"
	"fmt"

	"codeberg.org/snonux/goprecords/internal/storage"
)

// LoadAggregates reads all rows from the DB and builds Aggregates (same shape as file-based aggregation).
func LoadAggregates(ctx context.Context, db *sql.DB) (*Aggregates, error) {
	records, err := storage.LoadRecords(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load records: %w", err)
	}
	hostMeta, err := storage.LoadHostMeta(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load host meta: %w", err)
	}
	out := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	hostMaxBoot := make(map[string]uint64)
	hostLastKernel := make(map[string]string)

	for _, rec := range records {
		if rec.BootTime >= hostMaxBoot[rec.Host] {
			hostMaxBoot[rec.Host] = rec.BootTime
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
		if t, ok := hostMeta[host]; ok {
			h.LastUpdated = t
		}
	}
	return out, nil
}
