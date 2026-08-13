package storage

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"time"

	"codeberg.org/snonux/goprecords/internal/hostclass"
	"codeberg.org/snonux/goprecords/internal/recordline"
	"codeberg.org/snonux/goprecords/internal/recordsdir"
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
CREATE TABLE IF NOT EXISTS host_meta (
	host TEXT NOT NULL PRIMARY KEY,
	last_updated INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS host_class (
	host TEXT NOT NULL PRIMARY KEY,
	class TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS excluded_host (
	host TEXT NOT NULL PRIMARY KEY,
	reason TEXT NOT NULL DEFAULT '',
	excluded_at INTEGER NOT NULL DEFAULT (strftime('%s','now'))
);
`

// Record is one uptimed boot row stored in the record table.
type Record struct {
	Host        string
	Uptime      uint64
	BootTime    uint64
	OS          string
	KernelName  string
	KernelMajor string
}

// Open opens a SQLite database at path and verifies connectivity.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = OFF"); err != nil {
		db.Close()
		return nil, fmt.Errorf("pragma foreign_keys: %w", err)
	}
	return db, nil
}

// CreateSchema creates the record table and indexes if they do not exist.
func CreateSchema(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, schemaSQL)
	return err
}

// ResetRecords deletes all rows from the record table.
func ResetRecords(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM record")
	return err
}

// ResetHostMeta deletes all rows from the host_meta table.
func ResetHostMeta(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, "DELETE FROM host_meta")
	return err
}

// AddHostMeta inserts a host_meta row.
func AddHostMeta(ctx context.Context, tx *sql.Tx, host string, lastUpdated int64) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO host_meta (host, last_updated) VALUES (?, ?)", host, lastUpdated)
	if err != nil {
		return fmt.Errorf("insert host meta: %w", err)
	}
	return nil
}

// LoadHostMeta returns a map of host to last-updated time from the host_meta table.
func LoadHostMeta(ctx context.Context, db *sql.DB) (map[string]time.Time, error) {
	rows, err := db.QueryContext(ctx, "SELECT host, last_updated FROM host_meta")
	if err != nil {
		return nil, fmt.Errorf("query host meta: %w", err)
	}
	defer rows.Close()
	out := make(map[string]time.Time)
	for rows.Next() {
		var host string
		var lu int64
		if err := rows.Scan(&host, &lu); err != nil {
			return nil, fmt.Errorf("scan host meta: %w", err)
		}
		out[host] = time.Unix(lu, 0).UTC()
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows host meta: %w", err)
	}
	return out, nil
}

// SetHostClass stores the classification of a host, replacing a previous one.
func SetHostClass(ctx context.Context, db *sql.DB, host, class string) error {
	_, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO host_class (host, class) VALUES (?, ?)", host, class)
	if err != nil {
		return fmt.Errorf("set host class: %w", err)
	}
	return nil
}

// LoadHostClasses returns a map of host to classification name. Databases
// written by older versions have no host_class table; those yield an empty map
// so reports keep working until the next import recreates the schema.
func LoadHostClasses(ctx context.Context, db *sql.DB) (map[string]string, error) {
	ok, err := tableExists(ctx, db, "host_class")
	if err != nil || !ok {
		return map[string]string{}, err
	}
	rows, err := db.QueryContext(ctx, "SELECT host, class FROM host_class")
	if err != nil {
		return nil, fmt.Errorf("query host classes: %w", err)
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var host, class string
		if err := rows.Scan(&host, &class); err != nil {
			return nil, fmt.Errorf("scan host class: %w", err)
		}
		out[host] = class
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows host classes: %w", err)
	}
	return out, nil
}

func tableExists(ctx context.Context, db *sql.DB, name string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check table %s: %w", name, err)
	}
	return count > 0, nil
}

// ImportFromDir imports non-empty .records files from statsDir into the database,
// replacing existing rows. It is equivalent to ImportFromFS with os.DirFS(statsDir).
func ImportFromDir(ctx context.Context, db *sql.DB, statsDir string) error {
	return ImportFromFS(ctx, db, os.DirFS(statsDir))
}

// ImportFromFS reads non-empty .records files from the root of fsys into the database.
func ImportFromFS(ctx context.Context, db *sql.DB, fsys fs.FS) error {
	if err := ResetRecords(ctx, db); err != nil {
		return fmt.Errorf("reset records: %w", err)
	}
	if err := ResetHostMeta(ctx, db); err != nil {
		return fmt.Errorf("reset host meta: %w", err)
	}
	files, err := recordsdir.ListNonEmptyFilesFS(fsys, ".")
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
	for _, f := range files {
		if err := importFile(ctx, insert, fsys, f.Path, f.Host); err != nil {
			return err
		}
		if err := AddHostMeta(ctx, tx, f.Host, f.ModTime.Unix()); err != nil {
			return err
		}
	}
	if err := importHostClasses(ctx, tx, fsys); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// LoadRecords returns all rows from the record table ordered by host and boot time.
func LoadRecords(ctx context.Context, db *sql.DB) ([]Record, error) {
	var n int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM record").Scan(&n); err != nil {
		return nil, fmt.Errorf("count records: %w", err)
	}
	rows, err := db.QueryContext(ctx, "SELECT host, uptime_sec, boot_time, os, os_kernel_name, os_kernel_major FROM record ORDER BY host, boot_time")
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()
	out := make([]Record, 0, n)
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

// ExcludedHost holds an entry from the excluded_host table.
type ExcludedHost struct {
	Host       string
	Reason     string
	ExcludedAt int64
}

// AddExcludedHost inserts or replaces a host in the excluded_host table.
func AddExcludedHost(ctx context.Context, db *sql.DB, host, reason string) error {
	_, err := db.ExecContext(ctx,
		"INSERT OR REPLACE INTO excluded_host (host, reason) VALUES (?, ?)",
		host, reason)
	if err != nil {
		return fmt.Errorf("add excluded host: %w", err)
	}
	return nil
}

// RemoveExcludedHost removes a host from the excluded_host table.
func RemoveExcludedHost(ctx context.Context, db *sql.DB, host string) error {
	_, err := db.ExecContext(ctx, "DELETE FROM excluded_host WHERE host = ?", host)
	if err != nil {
		return fmt.Errorf("remove excluded host: %w", err)
	}
	return nil
}

// LoadExcludedHosts returns all rows from the excluded_host table.
func LoadExcludedHosts(ctx context.Context, db *sql.DB) ([]ExcludedHost, error) {
	rows, err := db.QueryContext(ctx, "SELECT host, reason, excluded_at FROM excluded_host ORDER BY host")
	if err != nil {
		return nil, fmt.Errorf("query excluded hosts: %w", err)
	}
	defer rows.Close()
	var out []ExcludedHost
	for rows.Next() {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		var e ExcludedHost
		if err := rows.Scan(&e.Host, &e.Reason, &e.ExcludedAt); err != nil {
			return nil, fmt.Errorf("scan excluded host: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows excluded hosts: %w", err)
	}
	return out, nil
}

// IsExcludedHost reports whether a host is in the excluded_host table.
func IsExcludedHost(ctx context.Context, db *sql.DB, host string) (bool, error) {
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM excluded_host WHERE host = ?", host).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check excluded host: %w", err)
	}
	return count > 0, nil
}

// importHostClasses mirrors the HOST.class files of the stats directory into
// the host_class table. Hosts without a .class file keep whatever the table
// already holds, so a classification set directly in the database survives
// re-imports.
func importHostClasses(ctx context.Context, tx *sql.Tx, fsys fs.FS) error {
	classes, err := hostclass.LoadFS(fsys, ".")
	if err != nil {
		return fmt.Errorf("read host classes: %w", err)
	}
	for host, c := range classes {
		_, err := tx.ExecContext(ctx,
			"INSERT OR REPLACE INTO host_class (host, class) VALUES (?, ?)", host, c.String())
		if err != nil {
			return fmt.Errorf("insert host class: %w", err)
		}
	}
	return nil
}

func importFile(ctx context.Context, insert *sql.Stmt, fsys fs.FS, relPath, host string) error {
	f, err := fsys.Open(relPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", relPath, err)
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
		return fmt.Errorf("scan %s: %w", relPath, err)
	}
	return nil
}
