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
	excluded, err := loadExcludedSet(ctx, dbPath)
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
		isExcluded := excluded[e.Host]
		writeHostMetric(&sb, e.Host, mtime, isExcluded)
	}
	return []byte(sb.String()), nil
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

func loadExcludedSet(ctx context.Context, dbPath string) (map[string]bool, error) {
	if dbPath == "" {
		return map[string]bool{}, nil
	}
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		return map[string]bool{}, nil
	}
	defer db.Close()
	return loadExcludedHosts(ctx, db)
}

func loadExcludedHosts(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	hosts, err := storage.LoadExcludedHosts(ctx, db)
	if err != nil {
		return nil, err
	}
	set := make(map[string]bool, len(hosts))
	for _, h := range hosts {
		set[h.Host] = true
	}
	return set, nil
}
