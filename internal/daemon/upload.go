package daemon

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/snonux/goprecords/internal/authkeys"
	"github.com/snonux/goprecords/internal/hostclass"
)

const maxUploadBytes = 8 << 20

// maxClassBytes caps a classification upload; a valid body is one short word.
const maxClassBytes = 64

func uploadKindExtension(kind string) (ext string, ok bool) {
	switch kind {
	case "txt":
		return ".txt", true
	case "cur.txt":
		return ".cur.txt", true
	case "records":
		return ".records", true
	case "os.txt":
		return ".os.txt", true
	case "cpuinfo.txt":
		return ".cpuinfo.txt", true
	case "class":
		return hostclass.Ext, true
	default:
		return "", false
	}
}

func uploadHandler(statsDir string, store *authkeys.Store) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveUploadPut(w, r, statsDir, store)
	})
}

func serveUploadPut(w http.ResponseWriter, r *http.Request, statsDir string, store *authkeys.Store) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	host, kind, ok := parseUploadPath(r.URL.Path)
	if !ok {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	ext, ok := uploadKindExtension(kind)
	if !ok {
		http.Error(w, "unknown file kind", http.StatusBadRequest)
		return
	}
	ctx := r.Context()
	nKeys, err := store.KeyCount(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nKeys > 0 && !uploadAuthorized(ctx, w, store, host, r.Header.Get("Authorization")) {
		return
	}
	rel := host + ext
	target := filepath.Join(statsDir, rel)
	if !fileUnderDir(statsDir, target) {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	if err := writeUpload(target, kind, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// writeUpload stores one uploaded file. Classifications are validated and
// normalized first, so a HOST.class file always holds a name the reports and
// the import understand.
func writeUpload(target, kind string, body io.Reader) error {
	if kind != "class" {
		return writeUploadBody(target, body)
	}
	raw, err := io.ReadAll(io.LimitReader(body, maxClassBytes))
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	c, err := hostclass.Parse(string(raw))
	if err != nil {
		return err
	}
	return writeUploadBody(target, strings.NewReader(c.String()+"\n"))
}

func uploadAuthorized(ctx context.Context, w http.ResponseWriter, store *authkeys.Store, host, authz string) bool {
	tok, ok := parseBearer(authz)
	if !ok || tok == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="upload"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	valid, err := store.Verify(ctx, host, tok)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return false
	}
	if !valid {
		http.Error(w, "forbidden", http.StatusForbidden)
		return false
	}
	return true
}

func parseUploadPath(path string) (host, kind string, ok bool) {
	const prefix = "/upload/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" || strings.Contains(rest, "..") {
		return "", "", false
	}
	i := strings.IndexByte(rest, '/')
	if i <= 0 || i >= len(rest)-1 {
		return "", "", false
	}
	host = rest[:i]
	kind = rest[i+1:]
	if !safeHostSegment(host) || strings.Contains(kind, "/") {
		return "", "", false
	}
	return host, kind, true
}

func safeHostSegment(s string) bool {
	if s == "" || len(s) > 253 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9':
		case c == '.' || c == '-' || c == '_':
		default:
			return false
		}
	}
	return true
}

func parseBearer(h string) (token string, ok bool) {
	h = strings.TrimSpace(h)
	const prefix = "Bearer "
	if len(h) < len(prefix) {
		return "", false
	}
	if !strings.EqualFold(h[:len(prefix)], prefix) {
		return "", false
	}
	t := strings.TrimSpace(h[len(prefix):])
	return t, t != ""
}

func fileUnderDir(dir, file string) bool {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	absFile, err := filepath.Abs(file)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absDir, absFile)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}

func writeUploadBody(path string, body io.Reader) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	lr := &io.LimitedReader{R: body, N: maxUploadBytes + 1}
	n, err := io.Copy(f, lr)
	if err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write: %w", err)
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("close temp: %w", err)
	}
	if n > maxUploadBytes {
		os.Remove(tmp)
		return fmt.Errorf("body too large")
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

func openAuthStore(ctx context.Context, statsDir, authDB string) (*authkeys.Store, error) {
	path := authDB
	if path == "" {
		path = authkeys.DefaultPath(statsDir)
	}
	s, err := authkeys.OpenStore(ctx, path)
	if err != nil {
		return nil, err
	}
	if err := s.EnsureSchema(ctx); err != nil {
		s.Close()
		return nil, err
	}
	return s, nil
}
