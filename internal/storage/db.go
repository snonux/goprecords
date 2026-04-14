package storage

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/goprecords/internal/recordline"
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

type Record struct {
	Host        string
	Uptime      uint64
	BootTime    uint64
	OS          string
	KernelName  string
	KernelMajor string
}

func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func CreateSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaSQL)
	return err
}

func ResetRecords(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM record")
	return err
}

func ImportFromDir(ctx context.Context, db *sql.DB, statsDir string) error {
	if err := ResetRecords(ctx, db); err != nil {
		return fmt.Errorf("reset records: %w", err)
	}
	entries, err := os.ReadDir(statsDir)
	if err != nil {
		return fmt.Errorf("read dir: %w", err)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	insert, err := tx.PrepareContext(ctx, "INSERT INTO record (host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major) VALUES (?, ?, ?, ?, ?, ?)")
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
		if err := importFile(ctx, insert, path, host); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

func LoadRecords(ctx context.Context, db *sql.DB) ([]Record, error) {
	rows, err := db.QueryContext(ctx, "SELECT host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major FROM record ORDER BY host, boot_time")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var rec Record
		var uptimeSec, bootTime int64
		if err := rows.Scan(&rec.Host, &uptimeSec, &bootTime, &rec.OS, &rec.KernelName, &rec.KernelMajor); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		rec.Uptime = uint64(uptimeSec)
		rec.BootTime = uint64(bootTime)
		out = append(out, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows: %w", err)
	}
	return out, nil
}

func importFile(ctx context.Context, insert *sql.Stmt, path, host string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec, ok := recordline.Parse(sc.Text())
		if !ok {
			continue
		}
		if _, err := insert.ExecContext(ctx, host, rec.Uptime, rec.BootTime, rec.OS, rec.KernelName, rec.KernelMajor); err != nil {
			return fmt.Errorf("insert: %w", err)
		}
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}
	return nil
}
