package daemon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/goprecords/internal/storage"
)

func TestMetricsEndpointEmpty(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(testHandler(t, statsDir))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	ct := res.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("Content-Type %q", ct)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "goprecords_host_records_last_update_timestamp_seconds") {
		t.Fatalf("missing metric name in body: %q", body)
	}
}

func TestMetricsEndpointWithHost(t *testing.T) {
	statsDir := t.TempDir()
	recordsFile := filepath.Join(statsDir, "myhost.records")
	if err := os.WriteFile(recordsFile, []byte("1:1:Linux 5.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(testHandler(t, statsDir))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	s := string(body)
	if !strings.Contains(s, `host="myhost"`) {
		t.Fatalf("missing host label: %q", s)
	}
	if !strings.Contains(s, `excluded="false"`) {
		t.Fatalf("missing excluded=false label: %q", s)
	}
}

func TestMetricsEndpointWithExcludedHost(t *testing.T) {
	statsDir := t.TempDir()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	recordsFile := filepath.Join(statsDir, "exchost.records")
	if err := os.WriteFile(recordsFile, []byte("1:1:Linux 5.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	db, err := storage.Open(ctx, dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.CreateSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	if err := storage.AddExcludedHost(ctx, db, "exchost", "test"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	h := metricsHandler(statsDir, dbPath)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, `excluded="true"`) {
		t.Fatalf("expected excluded=true for exchost: %q", body)
	}
}

func TestMetricsMethodNotAllowed(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(testHandler(t, statsDir))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/metrics", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d want 405", res.StatusCode)
	}
}

func TestBuildMetricsNoDBPath(t *testing.T) {
	statsDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(statsDir, "h1.records"), []byte("1:1:Linux 5.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body, err := buildMetrics(context.Background(), statsDir, "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(body)
	if !strings.Contains(s, `host="h1"`) {
		t.Fatalf("missing h1 in output: %q", s)
	}
	if !strings.Contains(s, `excluded="false"`) {
		t.Fatalf("missing excluded=false: %q", s)
	}
}
