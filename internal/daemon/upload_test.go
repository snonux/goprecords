package daemon

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"
)

func TestParseBearer(t *testing.T) {
	tests := []struct {
		name    string
		header  string
		wantTok string
		wantOK  bool
	}{
		{name: "valid", header: "Bearer abc.def", wantTok: "abc.def", wantOK: true},
		{name: "valid case", header: "bearer xyz", wantTok: "xyz", wantOK: true},
		{name: "extra space", header: "Bearer   tok  ", wantTok: "tok", wantOK: true},
		{name: "empty", header: "", wantOK: false},
		{name: "whitespace only", header: "   ", wantOK: false},
		{name: "too short", header: "Bear", wantOK: false},
		{name: "wrong scheme", header: "Basic xxx", wantOK: false},
		{name: "bearer no token", header: "Bearer ", wantOK: false},
		{name: "bearer only spaces", header: "Bearer    ", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tok, ok := parseBearer(tt.header)
			if ok != tt.wantOK || tok != tt.wantTok {
				t.Fatalf("parseBearer(%q) = (%q, %v) want (%q, %v)", tt.header, tok, ok, tt.wantTok, tt.wantOK)
			}
		})
	}
}

func TestParseUploadPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantHost string
		wantKind string
		wantOK   bool
	}{
		{name: "ok txt", path: "/upload/host1/txt", wantHost: "host1", wantKind: "txt", wantOK: true},
		{name: "ok nested kind", path: "/upload/my-host/cur.txt", wantHost: "my-host", wantKind: "cur.txt", wantOK: true},
		{name: "no prefix", path: "/x/upload/h/k", wantOK: false},
		{name: "empty rest", path: "/upload/", wantOK: false},
		{name: "dotdot", path: "/upload/../x/txt", wantOK: false},
		{name: "no slash", path: "/upload/only", wantOK: false},
		{name: "slash end", path: "/upload/host/", wantOK: false},
		{name: "bad host", path: "/upload/bad host/txt", wantOK: false},
		{name: "kind slash", path: "/upload/h/a/b", wantOK: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, k, ok := parseUploadPath(tt.path)
			if ok != tt.wantOK {
				t.Fatalf("ok=%v want %v (host=%q kind=%q)", ok, tt.wantOK, h, k)
			}
			if tt.wantOK && (h != tt.wantHost || k != tt.wantKind) {
				t.Fatalf("host=%q kind=%q want %q %q", h, k, tt.wantHost, tt.wantKind)
			}
		})
	}
}

func TestSafeHostSegment(t *testing.T) {
	tests := []struct {
		s    string
		want bool
	}{
		{"a", true},
		{"Host-1._x", true},
		{"", false},
		{"bad space", false},
		{"bad:colon", false},
		{strings.Repeat("a", 254), false},
		{strings.Repeat("a", 253), true},
	}
	for _, tt := range tests {
		if got := safeHostSegment(tt.s); got != tt.want {
			t.Fatalf("safeHostSegment(%q)=%v want %v", tt.s, got, tt.want)
		}
	}
}

func TestFileUnderDir(t *testing.T) {
	dir := t.TempDir()
	inside := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(inside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "escape.txt")
	if err := os.WriteFile(outside, []byte("y"), 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		dir  string
		file string
		want bool
	}{
		{"inside", dir, inside, true},
		{"same as dir", dir, dir, false},
		{"parent escape", dir, outside, false},
		{"dotdot file", dir, filepath.Join(dir, "..", filepath.Base(outside)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fileUnderDir(tt.dir, tt.file); got != tt.want {
				t.Fatalf("fileUnderDir(%q,%q)=%v want %v", tt.dir, tt.file, got, tt.want)
			}
		})
	}
}

func TestWriteUploadBodyTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.txt")
	body := bytes.Repeat([]byte{'z'}, maxUploadBytes+1)
	err := writeUploadBody(path, bytes.NewReader(body))
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Fatalf("expected body too large, got %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("final file should not exist")
	}
}

func TestWriteUploadBodyReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e.txt")
	err := writeUploadBody(path, iotest.ErrReader(errors.New("read boom")))
	if err == nil || !strings.Contains(err.Error(), "write") {
		t.Fatalf("expected write wrap error, got %v", err)
	}
}

func TestWriteUploadBodyCreateFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()
	path := filepath.Join(dir, "nope.txt")
	err := writeUploadBody(path, strings.NewReader("a"))
	if err == nil || !strings.Contains(err.Error(), "create temp") {
		t.Fatalf("expected create temp error, got %v", err)
	}
}

func TestWriteUploadBodyRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "block")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "block")
	err := writeUploadBody(path, strings.NewReader("x"))
	if err == nil || !strings.Contains(err.Error(), "rename") {
		t.Fatalf("expected rename error, got %v", err)
	}
}

func TestOpenAuthStoreBadPath(t *testing.T) {
	ctx := context.Background()
	_, err := openAuthStore(ctx, t.TempDir(), "/dev/null/impossible/goprecords-auth.db")
	if err == nil {
		t.Fatal("expected error opening auth store under /dev/null")
	}
}

func TestUploadMethodNotAllowedTable(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(testHandler(t, statsDir))
	defer srv.Close()
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			req, _ := http.NewRequest(method, srv.URL+"/upload/h/txt", nil)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != http.StatusMethodNotAllowed {
				t.Fatalf("status %d", res.StatusCode)
			}
		})
	}
}

func TestUploadAuthBearerNegativeTable(t *testing.T) {
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
	url := srv.URL + "/upload/myhost/txt"
	tests := []struct {
		name   string
		hdr    string
		status int
	}{
		{"no auth", "", http.StatusUnauthorized},
		{"basic", "Basic x", http.StatusUnauthorized},
		{"bearer empty", "Bearer ", http.StatusUnauthorized},
		{"wrong token", "Bearer not-the-token", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodPut, url, strings.NewReader("data"))
			if tt.hdr != "" {
				req.Header.Set("Authorization", tt.hdr)
			}
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			res.Body.Close()
			if res.StatusCode != tt.status {
				t.Fatalf("status %d want %d", res.StatusCode, tt.status)
			}
		})
	}
}
