package recordsdir

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestHostFromFileName(t *testing.T) {
	tests := []struct {
		file string
		want string
	}{
		{"earth.records", "earth"},
		{"myhost.example.com.records", "myhost"},
		{"single.records", "single"},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			if got := HostFromFileName(tt.file); got != tt.want {
				t.Fatalf("HostFromFileName(%q) = %q, want %q", tt.file, got, tt.want)
			}
		})
	}
}

func TestListNonEmptyFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "skip.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "empty.records"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "h1.records"), []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "nested.records"), 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := ListNonEmptyFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("len(entries) = %d, want 1: %#v", len(entries), entries)
	}
	if entries[0].Host != "h1" || filepath.Base(entries[0].Path) != "h1.records" {
		t.Fatalf("unexpected entry: %#v", entries[0])
	}
}

func TestListNonEmptyFiles_ReadError(t *testing.T) {
	_, err := ListNonEmptyFiles(filepath.Join(t.TempDir(), "nonexistent-subdir-nope"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestListNonEmptyFilesFS_MapFS(t *testing.T) {
	m := fstest.MapFS{
		"skip.txt":      &fstest.MapFile{Data: []byte("x"), Mode: 0o644},
		"empty.records": &fstest.MapFile{Data: nil, Mode: 0o644},
		"h1.records":    &fstest.MapFile{Data: []byte("line\n"), Mode: 0o644},
	}
	entries, err := ListNonEmptyFilesFS(m, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Host != "h1" || entries[0].Path != "h1.records" {
		t.Fatalf("got %#v", entries)
	}
}
