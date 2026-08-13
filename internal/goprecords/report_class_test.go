package goprecords

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"

	"codeberg.org/snonux/goprecords/internal/hostclass"
)

func classTestAggregates(classes map[string]hostclass.Class) *Aggregates {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	for host, c := range classes {
		h := NewHostAggregate(host, "Linux 5.10")
		h.Stats.Uptime = 86400000
		h.Stats.Boots = 10
		h.Stats.FirstBoot = 1000
		h.Stats.LastSeen = 86401000
		h.Class = c
		aggs.Host[host] = h
	}
	return aggs
}

func TestHostReportShowsClassColumn(t *testing.T) {
	tests := []struct {
		class hostclass.Class
		want  string
	}{
		{hostclass.Server, "S"},
		{hostclass.Workstation, "W"},
		{hostclass.Hybrid, "H"},
		{hostclass.Unknown, "U"},
	}
	for _, tt := range tests {
		t.Run(tt.class.String(), func(t *testing.T) {
			aggs := classTestAggregates(map[string]hostclass.Class{"host1": tt.class})
			for _, f := range []OutputFormat{FormatPlaintext, FormatMarkdown, FormatGemtext, FormatHTML} {
				report := NewHostReporter(aggs, 20, MetricUptime, f, 1).Report()
				if !strings.Contains(report, "Cls") {
					t.Fatalf("%s report has no Cls column:\n%s", f, report)
				}
				if !strings.Contains(report, classLegend) && f != FormatHTML {
					t.Fatalf("%s report has no class legend:\n%s", f, report)
				}
				if !strings.Contains(report, tt.want) {
					t.Fatalf("%s report misses class %q:\n%s", f, tt.want, report)
				}
			}
		})
	}
}

func TestHostReportDefaultsToUnknownClass(t *testing.T) {
	aggs := classTestAggregates(nil)
	h := NewHostAggregate("host1", "Linux 5.10")
	h.Stats.Uptime = 86400000
	h.Stats.LastSeen = 86401000
	aggs.Host["host1"] = h
	report := NewHostReporter(aggs, 20, MetricUptime, FormatPlaintext, 1).Report()
	if !strings.Contains(report, "| Cls |") || !strings.Contains(report, "  U |") {
		t.Fatalf("expected unknown class column, got:\n%s", report)
	}
}

func TestKernelReportHasNoClassColumn(t *testing.T) {
	aggs := classTestAggregates(nil)
	kernel := NewAggregate("Linux 5.10.0")
	kernel.Uptime = 86400000
	kernel.Boots = 5
	aggs.Kernel["Linux 5.10.0"] = kernel
	for _, f := range []OutputFormat{FormatPlaintext, FormatMarkdown, FormatGemtext, FormatHTML} {
		report := NewReporter(aggs, CategoryKernel, 20, MetricUptime, f, 1).Report()
		if strings.Contains(report, "Cls") {
			t.Fatalf("%s kernel report should have no Cls column:\n%s", f, report)
		}
	}
}

func TestAggregateReadsClassFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"earth.records": &fstest.MapFile{Data: []byte("86400:1000000:Linux 5.10.0-test\n")},
		"earth.class":   &fstest.MapFile{Data: []byte("server\n")},
		"mars.records":  &fstest.MapFile{Data: []byte("86400:1000000:Linux 5.10.0-test\n")},
	}
	aggregates, err := NewAggregatorFS(fsys).Aggregate(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got := aggregates.Host["earth"].Class; got != hostclass.Server {
		t.Errorf("class of earth = %v, want Server", got)
	}
	if got := aggregates.Host["mars"].Class; got != hostclass.Unknown {
		t.Errorf("class of mars = %v, want Unknown", got)
	}
}
