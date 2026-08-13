package hostclass

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    Class
		wantErr bool
	}{
		{in: "server", want: Server},
		{in: "SERVER", want: Server},
		{in: " s ", want: Server},
		{in: "workstation", want: Workstation},
		{in: "laptop", want: Workstation},
		{in: "W", want: Workstation},
		{in: "hybrid", want: Hybrid},
		{in: "h", want: Hybrid},
		{in: "unknown", want: Unknown},
		{in: "u", want: Unknown},
		{in: "", want: Unknown},
		{in: "router", wantErr: true},
		{in: "x", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Parse(%q) = %v, want error", tt.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Parse(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassStrings(t *testing.T) {
	tests := []struct {
		c     Class
		long  string
		short string
	}{
		{Server, "server", "S"},
		{Workstation, "workstation", "W"},
		{Hybrid, "hybrid", "H"},
		{Unknown, "unknown", "U"},
		{Class(42), "unknown", "U"},
	}
	for _, tt := range tests {
		if got := tt.c.String(); got != tt.long {
			t.Errorf("String() = %q, want %q", got, tt.long)
		}
		if got := tt.c.Short(); got != tt.short {
			t.Errorf("Short() = %q, want %q", got, tt.short)
		}
	}
}

func TestFileName(t *testing.T) {
	if got := FileName("earth"); got != "earth.class" {
		t.Fatalf("FileName = %q", got)
	}
}

func TestLoadFS(t *testing.T) {
	m := fstest.MapFS{
		"earth.class":              &fstest.MapFile{Data: []byte("server\n")},
		"myhost.example.com.class": &fstest.MapFile{Data: []byte("  Laptop  \n")},
		"commented.class":          &fstest.MapFile{Data: []byte("# a comment\nhybrid\n")},
		"broken.class":             &fstest.MapFile{Data: []byte("toaster\n")},
		"empty.class":              &fstest.MapFile{Data: nil},
		"ignored.txt":              &fstest.MapFile{Data: []byte("server\n")},
	}
	classes, err := LoadFS(m, ".")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]Class{
		"earth":     Server,
		"myhost":    Workstation,
		"commented": Hybrid,
	}
	if len(classes) != len(want) {
		t.Fatalf("got %#v, want %#v", classes, want)
	}
	for host, c := range want {
		if classes[host] != c {
			t.Errorf("class of %s = %v, want %v", host, classes[host], c)
		}
	}
}

func TestLoadFSHugeFileIgnored(t *testing.T) {
	m := fstest.MapFS{
		"big.class": &fstest.MapFile{Data: []byte(strings.Repeat("x", maxFileBytes*2))},
	}
	classes, err := LoadFS(m, ".")
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 0 {
		t.Fatalf("got %#v, want no classes", classes)
	}
}

func TestLoadFSReadDirError(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("expected error")
	}
}

func TestWriteFileAndLoad(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(dir, "earth", Hybrid); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "earth.class"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hybrid\n" {
		t.Fatalf("file content %q", b)
	}
	if err := WriteFile(dir, "earth", Server); err != nil {
		t.Fatal(err)
	}
	classes, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if classes["earth"] != Server {
		t.Fatalf("class of earth = %v, want Server", classes["earth"])
	}
}

func TestWriteFileUnwritableDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(dir, 0o755) }()
	if err := WriteFile(dir, "earth", Server); err == nil {
		t.Fatal("expected error writing into read-only dir")
	}
}
