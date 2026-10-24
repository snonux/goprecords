package daemon

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
)

// syncGateReader blocks after announcing that Read has started until release
// is closed, then delivers data. Used to force overlapping temp-file writes.
type syncGateReader struct {
	data    []byte
	pos     int
	started chan<- struct{}
	release <-chan struct{}
	once    sync.Once
}

func (g *syncGateReader) Read(p []byte) (int, error) {
	g.once.Do(func() { close(g.started) })
	<-g.release
	if g.pos >= len(g.data) {
		return 0, io.EOF
	}
	n := copy(p, g.data[g.pos:])
	g.pos += n
	return n, nil
}

func leftoverUploadTemps(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var left []string
	for _, e := range entries {
		name := e.Name()
		if strings.Contains(name, ".tmp") {
			left = append(left, name)
		}
	}
	return left
}

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
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps after oversize: %v", left)
	}
}

func TestWriteUploadBodyLongHostName(t *testing.T) {
	dir := t.TempDir()
	host := strings.Repeat("a", 246)
	if !safeHostSegment(host) {
		t.Fatalf("expected host len %d to be allowed by safeHostSegment", len(host))
	}
	path := filepath.Join(dir, host+".records")
	body := []byte("ok\n")
	if err := writeUploadBody(path, bytes.NewReader(body)); err != nil {
		t.Fatalf("writeUploadBody with long host: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(body) {
		t.Fatalf("body=%q want %q", got, body)
	}
	legacyPattern := "." + filepath.Base(path) + ".tmp-*"
	if _, err := os.CreateTemp(dir, legacyPattern); err == nil {
		t.Fatalf("expected legacy CreateTemp pattern %q to fail for this host length", legacyPattern)
	}
}

func TestWriteUploadBodyModeMatchesCreate(t *testing.T) {
	dir := t.TempDir()
	refPath := filepath.Join(dir, "ref")
	ref, err := os.Create(refPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ref.Close(); err != nil {
		t.Fatal(err)
	}
	refInfo, err := os.Stat(refPath)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, "out.records")
	if err := writeUploadBody(path, strings.NewReader("x\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != refInfo.Mode().Perm() {
		t.Fatalf("mode=%v want Create-like %v", info.Mode().Perm(), refInfo.Mode().Perm())
	}
}

func TestWriteUploadBodyConcurrentSamePath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.example.records")
	bodyA := bytes.Repeat([]byte{'A'}, 65536)
	bodyB := bytes.Repeat([]byte{'B'}, 65536)

	startedA := make(chan struct{})
	startedB := make(chan struct{})
	release := make(chan struct{})
	rA := &syncGateReader{data: bodyA, started: startedA, release: release}
	rB := &syncGateReader{data: bodyB, started: startedB, release: release}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = writeUploadBody(path, rA)
	}()
	go func() {
		defer wg.Done()
		errs[1] = writeUploadBody(path, rB)
	}()

	<-startedA
	<-startedB
	close(release)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, bodyA) && !bytes.Equal(got, bodyB) {
		t.Fatalf("final file mixed or truncated: len=%d distinct=%d", len(got), len(setOfBytes(got)))
	}
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps: %v", left)
	}
}

func TestWriteUploadBodyConcurrentDifferentHosts(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "host-a.records")
	pathB := filepath.Join(dir, "host-b.records")
	bodyA := bytes.Repeat([]byte{'A'}, 4096)
	bodyB := bytes.Repeat([]byte{'B'}, 4096)

	startedA := make(chan struct{})
	startedB := make(chan struct{})
	release := make(chan struct{})
	rA := &syncGateReader{data: bodyA, started: startedA, release: release}
	rB := &syncGateReader{data: bodyB, started: startedB, release: release}

	var wg sync.WaitGroup
	errs := make([]error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		errs[0] = writeUploadBody(pathA, rA)
	}()
	go func() {
		defer wg.Done()
		errs[1] = writeUploadBody(pathB, rB)
	}()
	<-startedA
	<-startedB
	close(release)
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("writer %d: %v", i, err)
		}
	}
	gotA, err := os.ReadFile(pathA)
	if err != nil {
		t.Fatal(err)
	}
	gotB, err := os.ReadFile(pathB)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(gotA, bodyA) || !bytes.Equal(gotB, bodyB) {
		t.Fatalf("independent hosts corrupted: a=%q b=%q", gotA[:1], gotB[:1])
	}
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps: %v", left)
	}
}

func setOfBytes(b []byte) map[byte]struct{} {
	m := make(map[byte]struct{}, 8)
	for _, c := range b {
		m[c] = struct{}{}
	}
	return m
}

func TestUploadConcurrentSameHostHTTP(t *testing.T) {
	statsDir := t.TempDir()
	srv := httptest.NewServer(testHandler(t, statsDir))
	defer srv.Close()

	bodyA := bytes.Repeat([]byte{'A'}, 65536)
	bodyB := bytes.Repeat([]byte{'B'}, 65536)
	startedA := make(chan struct{})
	startedB := make(chan struct{})
	release := make(chan struct{})

	type result struct {
		status int
		err    error
	}
	results := make([]result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/a.example/records",
			&syncGateReader{data: bodyA, started: startedA, release: release})
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			results[0] = result{err: err}
			return
		}
		res.Body.Close()
		results[0] = result{status: res.StatusCode}
	}()
	go func() {
		defer wg.Done()
		req, _ := http.NewRequest(http.MethodPut, srv.URL+"/upload/a.example/records",
			&syncGateReader{data: bodyB, started: startedB, release: release})
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			results[1] = result{err: err}
			return
		}
		res.Body.Close()
		results[1] = result{status: res.StatusCode}
	}()

	<-startedA
	<-startedB
	close(release)
	wg.Wait()

	for i, r := range results {
		if r.err != nil {
			t.Fatalf("request %d: %v", i, r.err)
		}
		if r.status != http.StatusNoContent {
			t.Fatalf("request %d status %d want 204", i, r.status)
		}
	}

	target := filepath.Join(statsDir, "a.example.records")
	first, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, bodyA) && !bytes.Equal(first, bodyB) {
		t.Fatalf("accepted payload mixed or truncated: len=%d", len(first))
	}
	later, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, later) {
		t.Fatal("accepted file changed after both uploads completed")
	}
	if left := leftoverUploadTemps(t, statsDir); len(left) != 0 {
		t.Fatalf("leftover temps: %v", left)
	}
}

func TestWriteUploadBodyReadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "e.txt")
	err := writeUploadBody(path, iotest.ErrReader(errors.New("read boom")))
	if err == nil || !strings.Contains(err.Error(), "write") {
		t.Fatalf("expected write wrap error, got %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("final file should not exist after read error")
	}
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps after read error: %v", left)
	}
}

func TestWriteUploadBodyCancelMidWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.txt")
	err := writeUploadBody(path, io.MultiReader(
		bytes.NewReader(bytes.Repeat([]byte{'x'}, 1024)),
		iotest.ErrReader(errors.New("canceled")),
	))
	if err == nil || !strings.Contains(err.Error(), "write") {
		t.Fatalf("expected write wrap error, got %v", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Fatal("final file should not exist after cancel")
	}
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps after cancel: %v", left)
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
	if left := leftoverUploadTemps(t, dir); len(left) != 0 {
		t.Fatalf("leftover temps after rename failure: %v", left)
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
	srv := httptest.NewServer(routes(statsDir, "", store, true))
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
