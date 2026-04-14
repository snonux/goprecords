package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"codeberg.org/snonux/goprecords/internal/version"
)

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found from test working directory")
		}
		dir = parent
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	fn()
	if err := w.Close(); err != nil {
		os.Stdout = old
		t.Fatal(err)
	}
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestStableVersionFlags(t *testing.T) {
	for _, arg := range []string{"-version", "--version"} {
		arg := arg
		t.Run(arg, func(t *testing.T) {
			out := captureStdout(t, func() {
				if err := Execute([]string{arg}); err != nil {
					t.Fatalf("Execute: %v", err)
				}
			})
			if !strings.Contains(out, version.Version) {
				t.Fatalf("stdout %q should contain version %q", out, version.Version)
			}
		})
	}
}

func TestStableReportFromFilesRequiresStatsDir(t *testing.T) {
	err := Execute(nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "stats-dir") {
		t.Fatalf("expected stats-dir in error, got %v", err)
	}
	err = Execute([]string{})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "stats-dir") {
		t.Fatalf("expected stats-dir in error, got %v", err)
	}
}

func TestStableReportFromFilesWithFixtures(t *testing.T) {
	root := moduleRoot(t)
	fixtures := filepath.Join(root, "fixtures")
	out := captureStdout(t, func() {
		if err := Execute([]string{"-stats-dir", fixtures}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "earth") {
		t.Fatalf("report output should mention fixture host earth; got len=%d", len(out))
	}
}

func TestStableImportAndQuery(t *testing.T) {
	root := moduleRoot(t)
	fixtures := filepath.Join(root, "fixtures")
	db := filepath.Join(t.TempDir(), "compat.db")
	if err := Execute([]string{"import", "-stats-dir", fixtures, "-db", db}); err != nil {
		t.Fatalf("import: %v", err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{"query", "-db", db, "-limit", "3"}); err != nil {
			t.Fatal(err)
		}
	})
	if len(strings.TrimSpace(out)) == 0 {
		t.Fatal("expected non-empty query output")
	}
}

func TestStableIntegrationTestSubcommand(t *testing.T) {
	root := moduleRoot(t)
	t.Chdir(root)
	if err := Execute([]string{"test"}); err != nil {
		t.Fatal(err)
	}
}

func TestStableDaemonRequiresStatsDir(t *testing.T) {
	t.Setenv("GOPRECORDS_STATS_DIR", "")
	for _, arg := range []string{"-daemon", "--daemon"} {
		arg := arg
		t.Run(arg, func(t *testing.T) {
			err := Execute([]string{arg})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), "stats-dir") {
				t.Fatalf("expected stats-dir in error, got %v", err)
			}
		})
	}
}

func TestStableSubcommandsStillRecognized(t *testing.T) {
	for _, sub := range []string{"import", "query", "test"} {
		if err := Execute([]string{sub}); err == nil {
			t.Fatalf("subcommand %q should fail without required args/env, not succeed silently", sub)
		}
	}
}
