package goprecords

import (
	"strings"
	"testing"
)

func TestNewReporter(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	reporter := NewReporter(aggs, CategoryHost, 20, MetricUptime, FormatPlaintext, 1)
	if reporter.limit != 20 {
		t.Errorf("expected limit 20, got %d", reporter.limit)
	}
	if reporter.category != CategoryHost {
		t.Errorf("expected CategoryHost, got %v", reporter.category)
	}
}

func TestNewHostReporter(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	reporter := NewHostReporter(aggs, 20, MetricUptime, FormatPlaintext, 1)
	if reporter.category != CategoryHost {
		t.Errorf("expected CategoryHost, got %v", reporter.category)
	}
}

func TestReportEmpty(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	reporter := NewReporter(aggs, CategoryHost, 20, MetricUptime, FormatPlaintext, 1)
	report := reporter.Report()
	if report != "" {
		t.Errorf("expected empty report for empty aggregates, got %q", report)
	}
}

func TestReportWithData(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	// Add a host
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.Uptime = 86400000
	hagg.Boots = 10
	hagg.FirstBoot = 1000
	hagg.LastSeen = 86401000
	aggs.Host["host1"] = hagg
	
	reporter := NewReporter(aggs, CategoryHost, 20, MetricUptime, FormatPlaintext, 1)
	report := reporter.Report()
	
	if report == "" {
		t.Error("expected non-empty report")
	}
	if !strings.Contains(report, "host1") {
		t.Error("expected report to contain host1")
	}
	if !strings.Contains(report, "Uptime") {
		t.Error("expected report to contain Uptime")
	}
}

func TestReportMarkdown(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.Uptime = 86400000
	hagg.Boots = 10
	hagg.FirstBoot = 1000
	hagg.LastSeen = 86401000
	aggs.Host["host1"] = hagg
	
	reporter := NewReporter(aggs, CategoryHost, 20, MetricUptime, FormatMarkdown, 2)
	report := reporter.Report()
	
	if !strings.Contains(report, "##") {
		t.Error("expected markdown header ##")
	}
	if !strings.Contains(report, "```") {
		t.Error("expected code block markers")
	}
}

func TestReportGemtext(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.Uptime = 86400000
	hagg.Boots = 10
	hagg.FirstBoot = 1000
	hagg.LastSeen = 86401000
	aggs.Host["host1"] = hagg
	
	reporter := NewReporter(aggs, CategoryHost, 20, MetricUptime, FormatGemtext, 2)
	report := reporter.Report()
	
	if !strings.Contains(report, "##") {
		t.Error("expected gemtext header ##")
	}
	if !strings.Contains(report, "```") {
		t.Error("expected code block markers")
	}
}

func TestReportMetrics(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.Uptime = 86400000
	hagg.Boots = 10
	hagg.FirstBoot = 1000
	hagg.LastSeen = 86401000
	aggs.Host["host1"] = hagg
	
	metrics := []Metric{MetricBoots, MetricUptime, MetricScore, MetricDowntime, MetricLifespan}
	for _, metric := range metrics {
		reporter := NewReporter(aggs, CategoryHost, 20, metric, FormatPlaintext, 1)
		report := reporter.Report()
		
		if report == "" {
			t.Errorf("expected non-empty report for metric %v", metric)
		}
	}
}

func TestReportKernelCategory(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	// Add kernel data
	kernel := NewAggregate("Linux 5.10.0")
	kernel.Uptime = 86400000
	kernel.Boots = 5
	aggs.Kernel["Linux 5.10.0"] = kernel
	
	reporter := NewReporter(aggs, CategoryKernel, 20, MetricUptime, FormatPlaintext, 1)
	report := reporter.Report()
	
	if report == "" {
		t.Error("expected non-empty report for Kernel category")
	}
	if !strings.Contains(report, "Linux 5.10.0") {
		t.Error("expected report to contain kernel name")
	}
}

func TestReportLimit(t *testing.T) {
	aggs := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	
	// Add multiple hosts
	for i := 0; i < 10; i++ {
		host := hostName(i)
		hagg := NewHostAggregate(host, "Linux")
		hagg.Uptime = uint64(86400000 * (10 - i))
		aggs.Host[host] = hagg
	}
	
	reporter := NewReporter(aggs, CategoryHost, 5, MetricUptime, FormatPlaintext, 1)
	report := reporter.Report()
	
	// Count entries (each entry line starts with |)
	lines := strings.Split(report, "\n")
	entryCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "|") && strings.Contains(line, ".") {
			entryCount++
		}
	}
	
	if entryCount > 5 {
		t.Errorf("expected at most 5 entries, got %d", entryCount)
	}
}

func hostName(i int) string {
	switch i {
	case 0:
		return "host0"
	case 1:
		return "host1"
	case 2:
		return "host2"
	case 3:
		return "host3"
	case 4:
		return "host4"
	case 5:
		return "host5"
	case 6:
		return "host6"
	case 7:
		return "host7"
	case 8:
		return "host8"
	default:
		return "host9"
	}
}
