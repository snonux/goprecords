package goprecords

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS record (
	host TEXT NOT NULL,
	uptime_sec INTEGER NOT NULL,
	boot_time INTEGER NOT NULL,
	os TEXT NOT NULL,
	os_kernel_name TEXT NOT NULL,
	os_kernel_major TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_record_host ON record(host);
CREATE INDEX IF NOT EXISTS idx_record_os ON record(os);
CREATE INDEX IF NOT EXISTS idx_record_os_kernel_name ON record(os_kernel_name);
CREATE INDEX IF NOT EXISTS idx_record_os_kernel_major ON record(os_kernel_major);
`

// OpenDB opens the SQLite database at path, creating the file if needed.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// CreateSchema creates the record table and indexes (idempotent).
func CreateSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaSQL)
	return err
}

// ResetRecords removes all rows so import is repeatable.
func ResetRecords(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM record")
	return err
}

// ImportFromDir reads all .records files from statsDir and inserts into the DB.
// Resets the record table first so the run is repeatable.
func ImportFromDir(ctx context.Context, db *sql.DB, statsDir string) error {
	if err := ResetRecords(ctx, db); err != nil {
		return fmt.Errorf("reset records: %w", err)
	}
	entries, err := os.ReadDir(statsDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	insert, err := db.PrepareContext(ctx, "INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer insert.Close()

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		path := filepath.Join(statsDir, e.Name())
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			continue
		}
		host := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if idx := strings.Index(host, "."); idx > 0 {
			host = host[:idx]
		}
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("open %s: %w", path, err)
		}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 3)
			if len(parts) != 3 {
				continue
			}
			uptimeSec, _ := strconv.ParseInt(parts[0], 10, 64)
			bootTime, _ := strconv.ParseInt(parts[1], 10, 64)
			osStr := parts[2]
			osKernelName := osStr
			if i := strings.Index(osStr, " "); i > 0 {
				osKernelName = osStr[:i]
			}
			osMajor := osKernelName + " "
			rest := osStr
			if i := strings.Index(osStr, " "); i >= 0 {
				rest = osStr[i+1:]
			}
			if j := strings.Index(rest, "."); j >= 0 {
				osMajor += rest[:j] + "..."
			} else {
				osMajor += rest + "..."
			}
			_, err := insert.ExecContext(ctx, host, uptimeSec, bootTime, osStr, osKernelName, osMajor)
			if err != nil {
				f.Close()
				return fmt.Errorf("insert: %w", err)
			}
		}
		f.Close()
		if err := sc.Err(); err != nil {
			return fmt.Errorf("scan %s: %w", path, err)
		}
	}
	return nil
}

// LoadAggregates reads all rows from the DB and builds Aggregates (same shape as file-based aggregation).
func LoadAggregates(ctx context.Context, db *sql.DB) (*Aggregates, error) {
	rows, err := db.QueryContext(ctx, "SELECT host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major FROM record ORDER BY host, boot_time")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	out := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	hostMaxBoot := make(map[string]int64)
	hostLastKernel := make(map[string]string)

	for rows.Next() {
		var host string
		var uptimeSec, bootTime int64
		var osStr, osKernelName, osKernelMajor string
		if err := rows.Scan(&host, &uptimeSec, &bootTime, &osStr, &osKernelName, &osKernelMajor); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		uptime := uint64(uptimeSec)
		boot := uint64(bootTime)
		if boot >= uint64(hostMaxBoot[host]) {
			hostMaxBoot[host] = int64(boot)
			hostLastKernel[host] = osStr
		}
		if _, ok := out.Host[host]; !ok {
			out.Host[host] = NewHostAggregate(host, "")
		}
		out.Host[host].AddRecord(uptime, boot)
		getOrNewAggregate(out.Kernel, osStr).AddRecord(uptime, boot)
		getOrNewAggregate(out.KernelName, osKernelName).AddRecord(uptime, boot)
		getOrNewAggregate(out.KernelMajor, osKernelMajor).AddRecord(uptime, boot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	for host, h := range out.Host {
		h.LastKernel = hostLastKernel[host]
	}
	return out, nil
}
