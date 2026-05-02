package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDarwinFishUploadClient(t *testing.T) {
	fish, err := exec.LookPath("fish")
	if err != nil {
		t.Skip("fish not installed")
	}

	dir := t.TempDir()
	binDir := filepath.Join(dir, "bin")
	bodyDir := filepath.Join(dir, "bodies")
	configDir := filepath.Join(dir, "config")
	tokenDir := filepath.Join(configDir, "goprecords-upload-mymac")
	for _, path := range []string{binDir, bodyDir, tokenDir} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	recordsPath := filepath.Join(dir, "records")
	if err := os.WriteFile(recordsPath, []byte("records-body\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tokenDir, "token"), []byte("secret-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	writeExecutable(t, filepath.Join(binDir, "curl"), `#!/bin/sh
for arg do
	if [ "$prev" = "--data-binary" ]; then data=${arg#@}; fi
	prev=$arg
	url=$arg
done
kind=${url##*/}
printf '%s\n' "$*" >> "$GOPRECORDS_TEST_LOG"
cp "$data" "$GOPRECORDS_TEST_BODY_DIR/$kind"
`)
	writeExecutable(t, filepath.Join(binDir, "uprecords"), `#!/bin/sh
if [ "$1" = "-a" ] && [ "$2" = "-m" ]; then
	printf 'all records\n'
else
	printf '%s\n' '-> current boot'
fi
`)
	writeExecutable(t, filepath.Join(binDir, "sw_vers"), `#!/bin/sh
printf 'ProductName: macOS\nProductVersion: 14.0\n'
`)
	writeExecutable(t, filepath.Join(binDir, "sysctl"), `#!/bin/sh
printf 'hw.model: MacBookPro\nhw.ncpu: 8\nhw.machine: arm64\n'
`)

	logPath := filepath.Join(dir, "curl.log")
	cmd := exec.Command(fish, "goprecords-upload-client-darwin.fish")
	cmd.Dir = filepath.Join("..", "..", "scripts")
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+dir,
		"XDG_CONFIG_HOME="+configDir,
		"GOPRECORDS_HOST=mymac",
		"GOPRECORDS_BASE_URL=https://example.test",
		"GOPRECORDS_RECORDS_FILE="+recordsPath,
		"GOPRECORDS_TEST_LOG="+logPath,
		"GOPRECORDS_TEST_BODY_DIR="+bodyDir,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fish client failed: %v\n%s", err, out)
	}

	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	logText := string(logData)
	if got := strings.Count(logText, "Authorization: Bearer secret-token"); got != 5 {
		t.Fatalf("Authorization header count = %d, want 5\n%s", got, logText)
	}
	for _, kind := range []string{"records", "txt", "cur.txt", "os.txt", "cpuinfo.txt"} {
		if !strings.Contains(logText, "https://example.test/upload/mymac/"+kind) {
			t.Fatalf("missing upload for %s\n%s", kind, logText)
		}
		if _, err := os.Stat(filepath.Join(bodyDir, kind)); err != nil {
			t.Fatalf("missing body for %s: %v", kind, err)
		}
	}
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}
