package daemon

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	b, _ := io.ReadAll(res.Body)
	if string(b) != "ok\n" {
		t.Fatalf("body %q", b)
	}
}

func TestHealthMethodNotAllowed(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/health", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestReport(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	h := Handler(fixtures)
	srv := httptest.NewServer(h)
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?category=Host&metric=Boots&limit=3&output-format=Plaintext")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Host") {
		t.Fatalf("expected report body, got %q", b)
	}
}

func TestReportBadQuery(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?category=Nope")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestRunEmptyStatsDir(t *testing.T) {
	err := Run(context.Background(), Config{StatsDir: "", Addr: ":0"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunEmptyAddr(t *testing.T) {
	err := Run(context.Background(), Config{StatsDir: t.TempDir(), Addr: ""})
	if err == nil {
		t.Fatal("expected error")
	}
}
