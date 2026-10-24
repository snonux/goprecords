package goprecords

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPrepareGoldenFixturesSetsContractModTime(t *testing.T) {
	src := filepath.Join("..", "..", "fixtures")
	earthSrc := filepath.Join(src, "earth.records")
	before, err := os.Stat(earthSrc)
	if err != nil {
		t.Fatal(err)
	}

	dir, err := prepareGoldenFixtures(src)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	after, err := os.Stat(earthSrc)
	if err != nil {
		t.Fatal(err)
	}
	if !before.ModTime().Equal(after.ModTime()) {
		t.Fatalf("canonical fixture mtime mutated: before %v after %v", before.ModTime(), after.ModTime())
	}

	got, err := os.Stat(filepath.Join(dir, "earth.records"))
	if err != nil {
		t.Fatal(err)
	}
	if !got.ModTime().UTC().Equal(fixtureGoldenModTime) {
		t.Fatalf("prepared earth.records mtime = %v, want %v", got.ModTime().UTC(), fixtureGoldenModTime)
	}
}

func TestLastUpdatedFollowsIntentionalRecordsModTime(t *testing.T) {
	dir := t.TempDir()
	content := []byte("86400:1000000:Linux 5.10.0-test\n")
	older := time.Date(2019, 5, 1, 10, 0, 0, 0, time.UTC)
	newer := time.Date(2022, 11, 20, 15, 45, 0, 0, time.UTC)
	writeRecordsWithModTime(t, dir, "oldhost.records", content, older)
	writeRecordsWithModTime(t, dir, "newhost.records", content, newer)

	aggs, err := NewAggregator(dir).Aggregate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := aggs.Host["oldhost"].LastUpdated.UTC(); !got.Equal(older) {
		t.Fatalf("oldhost LastUpdated = %v, want %v", got, older)
	}
	if got := aggs.Host["newhost"].LastUpdated.UTC(); !got.Equal(newer) {
		t.Fatalf("newhost LastUpdated = %v, want %v", got, newer)
	}

	report := NewHostReporter(aggs, 20, MetricUptime, FormatPlaintext, 1).Report()
	if !strings.Contains(report, "2019-05-01 10:00") {
		t.Fatalf("expected older intentional Updated in report, got %q", report)
	}
	if !strings.Contains(report, "2022-11-20 15:45") {
		t.Fatalf("expected newer intentional Updated in report, got %q", report)
	}
	if strings.Contains(report, fixtureGoldenModTime.UTC().Format("2006-01-02 15:04")) {
		t.Fatalf("control fixtures must not use golden contract time; got %q", report)
	}
}

func writeRecordsWithModTime(t *testing.T, dir, name string, content []byte, modTime time.Time) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
