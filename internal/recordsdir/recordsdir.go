package recordsdir

import (
	"os"
	"path/filepath"
	"strings"
)

type Entry struct {
	Path string
	Host string
}

func HostFromFileName(name string) string {
	host := strings.TrimSuffix(name, filepath.Ext(name))
	if idx := strings.Index(host, "."); idx > 0 {
		host = host[:idx]
	}
	return host
}

func ListNonEmptyFiles(dir string) ([]Entry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			continue
		}
		out = append(out, Entry{Path: path, Host: HostFromFileName(e.Name())})
	}
	return out, nil
}
