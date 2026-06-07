package recordsdir

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Entry struct {
	Path    string
	Host    string
	ModTime time.Time
}

func HostFromFileName(name string) string {
	host := strings.TrimSuffix(name, filepath.Ext(name))
	if idx := strings.Index(host, "."); idx > 0 {
		host = host[:idx]
	}
	return host
}

func listRecordsFileNames(fsys fs.FS, root string) ([]Entry, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		rel := path.Join(root, e.Name())
		info, err := fs.Stat(fsys, rel)
		if err != nil || info.Size() == 0 {
			continue
		}
		out = append(out, Entry{
			Path:    rel,
			Host:    HostFromFileName(e.Name()),
			ModTime: info.ModTime(),
		})
	}
	return out, nil
}

// ListNonEmptyFilesFS returns non-empty .records files under root within fsys.
func ListNonEmptyFilesFS(fsys fs.FS, root string) ([]Entry, error) {
	return listRecordsFileNames(fsys, root)
}

func ListNonEmptyFiles(dir string) ([]Entry, error) {
	entries, err := listRecordsFileNames(os.DirFS(dir), ".")
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entries[i].Path = filepath.Join(dir, filepath.Base(entries[i].Path))
	}
	return entries, nil
}
