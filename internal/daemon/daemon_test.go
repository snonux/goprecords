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
	if ct := res.Header.Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type %q", ct)
	}
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "Host") {
		t.Fatalf("expected report body, got %q", b)
	}
}

func TestReportQueryAliases(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	srv := httptest.NewServer(Handler(fixtures))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?Category=Host&Metric=Uptime&limit=2&OutputFormat=Markdown")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/markdown; charset=utf-8" {
		t.Fatalf("Content-Type %q", ct)
	}
	b, _ := io.ReadAll(res.Body)
	body := string(b)
	if !strings.Contains(body, "# Top") || !strings.Contains(body, "```") {
		t.Fatalf("expected markdown heading and fence, got %q", b)
	}
}

func TestReportGemtextContentType(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	srv := httptest.NewServer(Handler(fixtures))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?output-format=Gemtext&limit=2")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/gemini; charset=utf-8" {
		t.Fatalf("Content-Type %q", ct)
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
