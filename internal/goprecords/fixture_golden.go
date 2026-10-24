package goprecords

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var fixtureGoldenModTime = time.Date(2026, 3, 3, 18, 29, 0, 0, time.UTC)

func prepareGoldenFixtures(srcDir string) (string, error) {
	dst, err := os.MkdirTemp("", "goprecords-fixtures-*")
	if err != nil {
		return "", err
	}
	if err := copyTree(srcDir, dst); err != nil {
		_ = os.RemoveAll(dst)
		return "", fmt.Errorf("copy fixtures: %w", err)
	}
	if err := setRecordsModTimes(dst, fixtureGoldenModTime); err != nil {
		_ = os.RemoveAll(dst)
		return "", fmt.Errorf("set fixture mtimes: %w", err)
	}
	return dst, nil
}

func setRecordsModTimes(dir string, modTime time.Time) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if err := os.Chtimes(path, modTime, modTime); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
