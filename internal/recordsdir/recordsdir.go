package recordsdir

import (
	"io/fs"
	"os"
	"path"
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

func listRecordsFileNames(fsys fs.FS, root string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, root)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		rel := path.Join(root, e.Name())
		info, err := fs.Stat(fsys, rel)
		if err != nil || info.Size() == 0 {
			continue
		}
		names = append(names, e.Name())
	}
	return names, nil
}

// ListNonEmptyFilesFS returns non-empty .records files under root within fsys.
func ListNonEmptyFilesFS(fsys fs.FS, root string) ([]Entry, error) {
	names, err := listRecordsFileNames(fsys, root)
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, name := range names {
		out = append(out, Entry{
			Path: path.Join(root, name),
			Host: HostFromFileName(name),
		})
	}
	return out, nil
}

func ListNonEmptyFiles(dir string) ([]Entry, error) {
	names, err := listRecordsFileNames(os.DirFS(dir), ".")
	if err != nil {
		return nil, err
	}
	var out []Entry
	for _, name := range names {
		out = append(out, Entry{
			Path: filepath.Join(dir, name),
			Host: HostFromFileName(name),
		})
	}
	return out, nil
}
