// Package hostclass classifies hosts as server, workstation/laptop, hybrid or
// unknown. A classification is stored in a "HOST.class" file next to the
// uptimed record files, so it can be edited by hand in the stats directory or
// uploaded through the daemon API (PUT /upload/HOST/class).
package hostclass

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/snonux/goprecords/internal/recordsdir"
)

// Ext is the file name extension of a host classification file.
const Ext = ".class"

// maxFileBytes caps how much of a .class file is read; a valid file holds one
// short word, so anything bigger is treated as garbage.
const maxFileBytes = 256

// Class is the kind of machine a host is.
type Class int

const (
	// Unknown is the default when no classification was recorded.
	Unknown Class = iota
	Server
	Workstation
	Hybrid
)

// String returns the canonical long name, as stored in a .class file.
func (c Class) String() string {
	switch c {
	case Server:
		return "server"
	case Workstation:
		return "workstation"
	case Hybrid:
		return "hybrid"
	default:
		return "unknown"
	}
}

// Short returns the single letter shown in report tables.
func (c Class) Short() string {
	switch c {
	case Server:
		return "S"
	case Workstation:
		return "W"
	case Hybrid:
		return "H"
	default:
		return "U"
	}
}

// Parse parses a classification from its long or short form (case-insensitive).
// An empty string is Unknown; "laptop" is an alias for "workstation".
func Parse(s string) (Class, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "unknown", "u":
		return Unknown, nil
	case "server", "s":
		return Server, nil
	case "workstation", "laptop", "w":
		return Workstation, nil
	case "hybrid", "h":
		return Hybrid, nil
	default:
		return Unknown, fmt.Errorf("invalid host class %q", s)
	}
}

// FileName returns the .class file name for a host.
func FileName(host string) string {
	return host + Ext
}

// LoadFS returns the classification of every host with a .class file under root
// in fsys. Files whose content is not a valid class are ignored, so a typo in a
// hand-edited file leaves that host Unknown instead of failing every report.
func LoadFS(fsys fs.FS, root string) (map[string]Class, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}
	out := make(map[string]Class)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), Ext) {
			continue
		}
		c, ok := readClassFile(fsys, path.Join(root, e.Name()))
		if !ok {
			continue
		}
		out[recordsdir.HostFromFileName(e.Name())] = c
	}
	return out, nil
}

// Load returns the classification of every host with a .class file in dir.
func Load(dir string) (map[string]Class, error) {
	return LoadFS(os.DirFS(dir), ".")
}

func readClassFile(fsys fs.FS, relPath string) (Class, bool) {
	f, err := fsys.Open(relPath)
	if err != nil {
		return Unknown, false
	}
	defer f.Close()
	sc := bufio.NewScanner(io.LimitReader(f, maxFileBytes))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		c, err := Parse(line)
		if err != nil {
			return Unknown, false
		}
		return c, true
	}
	return Unknown, false
}

// WriteFile stores the classification of host in dir, replacing any previous
// one. It writes the canonical long name so the file stays hand-editable.
func WriteFile(dir, host string, c Class) error {
	target := filepath.Join(dir, FileName(host))
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, []byte(c.String()+"\n"), 0o644); err != nil {
		return fmt.Errorf("write class file: %w", err)
	}
	if err := os.Rename(tmp, target); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("rename class file: %w", err)
	}
	return nil
}
