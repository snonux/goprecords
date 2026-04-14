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

func TestReportHTTPTable(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	srv := httptest.NewServer(Handler(fixtures))
	defer srv.Close()
	tests := []struct {
		name       string
		query      string
		wantCode   int
		wantCTPfx  string
		bodyNeedle []string
	}{
		{
			name:       "plaintext",
			query:      "category=Host&metric=Boots&limit=3&output-format=Plaintext",
			wantCode:   http.StatusOK,
			wantCTPfx:  "text/plain",
			bodyNeedle: []string{"Host"},
		},
		{
			name:       "markdown aliases",
			query:      "Category=Host&Metric=Uptime&limit=2&OutputFormat=Markdown",
			wantCode:   http.StatusOK,
			wantCTPfx:  "text/markdown",
			bodyNeedle: []string{"# Top", "```"},
		},
		{
			name:       "html",
			query:      "OutputFormat=HTML&limit=2",
			wantCode:   http.StatusOK,
			wantCTPfx:  "text/html",
			bodyNeedle: []string{"<!DOCTYPE html>", "<pre>"},
		},
		{
			name:      "gemtext",
			query:     "output-format=Gemtext&limit=2",
			wantCode:  http.StatusOK,
			wantCTPfx: "text/gemini",
		},
		{
			name:     "bad category",
			query:    "category=Nope",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid limit",
			query:    "limit=notnum",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid all bool",
			query:    "all=nope",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "downtime on non host",
			query:    "category=Kernel&metric=Downtime&limit=2",
			wantCode: http.StatusBadRequest,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := http.Get(srv.URL + "/report?" + tt.query)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tt.wantCode {
				t.Fatalf("status %d want %d", res.StatusCode, tt.wantCode)
			}
			if tt.wantCTPfx != "" {
				ct := res.Header.Get("Content-Type")
				if !strings.HasPrefix(ct, tt.wantCTPfx) {
					t.Fatalf("Content-Type %q want prefix %q", ct, tt.wantCTPfx)
				}
			}
			b, _ := io.ReadAll(res.Body)
			body := string(b)
			for _, sub := range tt.bodyNeedle {
				if !strings.Contains(body, sub) {
					t.Fatalf("body missing %q: %q", sub, body)
				}
			}
		})
	}
}

func TestReportMethodNotAllowed(t *testing.T) {
	fixtures := filepath.Join("..", "..", "fixtures")
	srv := httptest.NewServer(Handler(fixtures))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/report?limit=2", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", res.StatusCode)
	}
}

func TestReportAggregateFailure(t *testing.T) {
	dir := t.TempDir()
	line := "1:1:Linux 5.13.14-200.fc34.x86_64\n"
	if err := os.WriteFile(filepath.Join(dir, "dup.x.records"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dup.y.records"), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(dir))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/report?limit=2")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status %d want 500", res.StatusCode)
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

func TestRunUsesStdoutWhenLogOutputNil(t *testing.T) {
	old := os.Stdout
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = pw
	var logBuf bytes.Buffer
	copyDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(&logBuf, pr)
		close(copyDone)
	}()
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{StatsDir: t.TempDir(), Addr: "127.0.0.1:0", LogOutput: nil}
	runDone := make(chan struct{})
	go func() {
		_ = Run(ctx, cfg)
		close(runDone)
	}()
	deadline := time.After(2 * time.Second)
	for !strings.Contains(logBuf.String(), "daemon_listen") {
		select {
		case <-deadline:
			cancel()
			_ = pw.Close()
			<-runDone
			<-copyDone
			_ = pr.Close()
			os.Stdout = old
			t.Fatalf("timeout, got %q", logBuf.String())
		case <-time.After(5 * time.Millisecond):
		}
	}
	cancel()
	<-runDone
	os.Stdout = old
	if err := pw.Close(); err != nil {
		t.Fatal(err)
	}
	<-copyDone
	if err := pr.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRunInvalidListenAddress(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := Run(ctx, Config{StatsDir: t.TempDir(), Addr: ":999999999"})
	if err == nil {
		t.Fatal("expected listen error")
	}
	if !strings.Contains(err.Error(), "listen") {
		t.Fatalf("expected listen in error: %v", err)
	}
}

func TestAccessLogImplicitOKStatus(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	mux := http.NewServeMux()
	mux.HandleFunc("/nohdr", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(withAccessLog(log, mux))
	defer srv.Close()
	res, err := http.Get(srv.URL + "/nohdr")
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if !strings.Contains(buf.String(), "status=200") {
		t.Fatalf("log %q", buf.String())
	}
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
