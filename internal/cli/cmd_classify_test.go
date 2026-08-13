package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassifyWritesClassFile(t *testing.T) {
	dir := t.TempDir()
	if err := Execute([]string{"classify", "-stats-dir=" + dir, "earth", "S"}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "earth.class"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "server\n" {
		t.Fatalf("class file content %q", b)
	}
}

func TestClassifyAlsoUpdatesDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := Execute([]string{"classify", "-stats-dir=" + dir, "-db=" + dbPath, "earth", "hybrid"}); err != nil {
		t.Fatal(err)
	}
	out := captureStdout(t, func() {
		if err := Execute([]string{"list-classes", "-db=" + dbPath}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "earth") || !strings.Contains(out, "hybrid") {
		t.Fatalf("list-classes output %q", out)
	}
}

func TestClassifyInvalidInput(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"missing args", []string{"classify", "-stats-dir=" + dir, "earth"}, "hostname or class"},
		{"missing target", []string{"classify", "earth", "server"}, "stats-dir"},
		{"invalid class", []string{"classify", "-stats-dir=" + dir, "earth", "toaster"}, "invalid host class"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Execute(tt.args)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("got %v, want error containing %q", err, tt.want)
			}
		})
	}
}

func TestListClasses(t *testing.T) {
	dir := t.TempDir()
	out := captureStdout(t, func() {
		if err := Execute([]string{"list-classes", "-stats-dir=" + dir}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "no classified hosts") {
		t.Fatalf("output %q", out)
	}
	if err := Execute([]string{"classify", "-stats-dir=" + dir, "t450", "laptop"}); err != nil {
		t.Fatal(err)
	}
	out = captureStdout(t, func() {
		if err := Execute([]string{"list-classes", "-stats-dir=" + dir}); err != nil {
			t.Fatal(err)
		}
	})
	if !strings.Contains(out, "t450") || !strings.Contains(out, "workstation") {
		t.Fatalf("output %q", out)
	}
	if err := Execute([]string{"list-classes"}); err == nil {
		t.Fatal("expected error without -stats-dir or -db")
	}
}
