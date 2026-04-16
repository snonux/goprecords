package daemon

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"codeberg.org/snonux/goprecords/internal/recordsdir"
	"codeberg.org/snonux/goprecords/internal/storage"
)

func metricsHandler(statsDir, dbPath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		body, err := buildMetrics(r.Context(), statsDir, dbPath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func buildMetrics(ctx context.Context, statsDir, dbPath string) ([]byte, error) {
	entries, err := recordsdir.ListNonEmptyFiles(statsDir)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}
	db := openMetricsDB(ctx, dbPath)
	if db != nil {
		defer db.Close()
	}
	excluded, err := loadExcludedMap(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load excluded hosts: %w", err)
	}
	var sb strings.Builder
	writeMetricHeader(&sb)
	for _, e := range entries {
		mtime, err := recordsMtime(statsDir, e.Host)
		if err != nil {
			continue
		}
		isExcluded := resolveExcluded(ctx, db, e.Host, mtime, excluded)
		writeHostMetric(&sb, e.Host, mtime, isExcluded)
	}
	return []byte(sb.String()), nil
}

// openMetricsDB opens (and schema-initialises) the SQLite DB at dbPath.
// Returns nil silently on empty path or any open/schema error so the metrics
// endpoint degrades gracefully when no DB is configured.
func openMetricsDB(ctx context.Context, dbPath string) *sql.DB {
	if dbPath == "" {
		return nil
	}
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		return nil
	}
	if err := storage.CreateSchema(ctx, db); err != nil {
		db.Close()
		return nil
	}
	return db
}

// loadExcludedMap returns a map of host→excluded_at for all excluded hosts.
func loadExcludedMap(ctx context.Context, db *sql.DB) (map[string]int64, error) {
	if db == nil {
		return map[string]int64{}, nil
	}
	hosts, err := storage.LoadExcludedHosts(ctx, db)
	if err != nil {
		return nil, err
	}
	m := make(map[string]int64, len(hosts))
	for _, h := range hosts {
		m[h.Host] = h.ExcludedAt
	}
	return m, nil
}

// resolveExcluded returns true if the host is currently excluded.
// If the records file mtime is newer than excluded_at the exclusion is
// automatically lifted in the DB and the host is treated as active again.
func resolveExcluded(ctx context.Context, db *sql.DB, host string, mtime int64, excluded map[string]int64) bool {
	excludedAt, isExcluded := excluded[host]
	if !isExcluded {
		return false
	}
	if mtime > excludedAt {
		if db != nil {
			_ = storage.RemoveExcludedHost(ctx, db, host)
		}
		return false
	}
	return true
}

func writeMetricHeader(sb *strings.Builder) {
	sb.WriteString("# HELP goprecords_host_records_last_update_timestamp_seconds Unix timestamp of the last records file update for each host.\n")
	sb.WriteString("# TYPE goprecords_host_records_last_update_timestamp_seconds gauge\n")
}

func writeHostMetric(sb *strings.Builder, host string, mtime int64, excluded bool) {
	excludedVal := "false"
	if excluded {
		excludedVal = "true"
	}
	fmt.Fprintf(sb, "goprecords_host_records_last_update_timestamp_seconds{host=%q,excluded=%q} %d\n",
		host, excludedVal, mtime)
}

func recordsMtime(statsDir, host string) (int64, error) {
	path := filepath.Join(statsDir, host+".records")
	fi, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return fi.ModTime().Unix(), nil
}
