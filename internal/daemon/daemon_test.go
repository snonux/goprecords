package daemon

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestLivez(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/livez")
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

func TestReadyzOK(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/readyz")
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

func TestReadyzMethodNotAllowed(t *testing.T) {
	srv := httptest.NewServer(Handler(t.TempDir()))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/readyz", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestReadyzMissingStatsDir(t *testing.T) {
	statsDir := filepath.Join(t.TempDir(), "absent")
	srv := httptest.NewServer(readiness(statsDir, ""))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d want 503", res.StatusCode)
	}
}

func TestReadyzStatsDirNotDirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(readiness(f, ""))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d want 503", res.StatusCode)
	}
}

func TestReadyzStatsDirNotWritable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()
	srv := httptest.NewServer(readiness(dir, ""))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d want 503", res.StatusCode)
	}
}

func TestReadyzAuthDBDirNotWritable(t *testing.T) {
	statsDir := t.TempDir()
	authDir := t.TempDir()
	authDB := filepath.Join(authDir, "auth.db")
	if err := os.Chmod(authDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(authDir, 0o755) }()
	srv := httptest.NewServer(readiness(statsDir, authDB))
	defer srv.Close()
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d want 503", res.StatusCode)
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

func TestReportHTMLContentType(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	srv := httptest.NewServer(Handler(fixtures))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?OutputFormat=HTML&limit=2")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("Content-Type %q", ct)
	}
	b, _ := io.ReadAll(res.Body)
	body := string(b)
	if !strings.Contains(body, "<!DOCTYPE html>") || !strings.Contains(body, "<pre>") {
		t.Fatalf("expected HTML body, got %q", body)
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

func TestRunWritesDaemonListenToLogOutput(t *testing.T) {
	var buf bytes.Buffer
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{StatsDir: t.TempDir(), Addr: "127.0.0.1:0", LogOutput: &buf}
	done := make(chan struct{})
	go func() {
		_ = Run(ctx, cfg)
		close(done)
	}()
	deadline := time.After(2 * time.Second)
	for !strings.Contains(buf.String(), "daemon_listen") {
		select {
		case <-deadline:
			t.Fatalf("timeout waiting for daemon_listen, got %q", buf.String())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-done
}

func TestAccessLogLineToWriter(t *testing.T) {
	var buf bytes.Buffer
	h := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	log := slog.New(h)
	statsDir := t.TempDir()
	store, err := openAuthStore(context.Background(), statsDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	srv := httptest.NewServer(withAccessLog(log, routes(statsDir, "", store)))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	body := buf.String()
	if !strings.Contains(body, "http_request") || !strings.Contains(body, "method=GET") {
		t.Fatalf("expected http_request line with method=GET, got %q", body)
	}
	if !strings.Contains(body, "path=/health") || !strings.Contains(body, "status=200") {
		t.Fatalf("expected path and status in log, got %q", body)
	}
}

func TestUploadOpenWhenNoKeys(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(Handler(statsDir))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/myhost/txt", strings.NewReader("hello"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", res.StatusCode)
	}
	b, err := os.ReadFile(filepath.Join(statsDir, "myhost.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hello" {
		t.Fatalf("file %q", b)
	}
}

func TestUploadRequiresBearerWhenKeysExist(t *testing.T) {
	statsDir := t.TempDir()
	ctx := context.Background()
	store, err := openAuthStore(ctx, statsDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.CreateKey(ctx, "myhost"); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(routes(statsDir, "", store))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/myhost/txt", strings.NewReader("x"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d want 401", res.StatusCode)
	}
}

func TestUploadWithValidBearer(t *testing.T) {
	statsDir := t.TempDir()
	ctx := context.Background()
	store, err := openAuthStore(ctx, statsDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tok, err := store.CreateKey(ctx, "myhost")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(routes(statsDir, "", store))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/myhost/os.txt", strings.NewReader("os"))
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestUploadWrongHostForbidden(t *testing.T) {
	statsDir := t.TempDir()
	ctx := context.Background()
	store, err := openAuthStore(ctx, statsDir, "")
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tok, err := store.CreateKey(ctx, "myhost")
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(routes(statsDir, "", store))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/other/txt", strings.NewReader("x"))
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d want 403", res.StatusCode)
	}
}

func TestUploadBadKind(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(Handler(statsDir))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/myhost/nope", strings.NewReader("x"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestUploadAllKindsWriteExpectedFiles(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(Handler(statsDir))
	defer srv.Close()
	cases := []struct {
		kind     string
		wantName string
		body     string
	}{
		{"txt", "myhost.txt", "a"},
		{"cur.txt", "myhost.cur.txt", "b"},
		{"records", "myhost.records", "c"},
		{"os.txt", "myhost.os.txt", "d"},
		{"cpuinfo.txt", "myhost.cpuinfo.txt", "e"},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			url := srv.URL + "/upload/myhost/" + tc.kind
			req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader(tc.body))
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != http.StatusNoContent {
				t.Fatalf("status %d", res.StatusCode)
			}
			b, err := os.ReadFile(filepath.Join(statsDir, tc.wantName))
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != tc.body {
				t.Fatalf("file %s: got %q want %q", tc.wantName, b, tc.body)
			}
		})
	}
}
