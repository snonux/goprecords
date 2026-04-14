package goprecords

import (
	"bytes"
	"net/url"
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
	plain, ok := reporter.(*PlaintextReporter)
	if !ok {
		t.Fatalf("expected PlaintextReporter, got %T", reporter)
	}
	if plain.builder.limit != 20 {
		t.Errorf("expected limit 20, got %d", plain.builder.limit)
	}
	if plain.builder.category != CategoryHost {
		t.Errorf("expected CategoryHost, got %v", plain.builder.category)
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
	plain, ok := reporter.(*PlaintextReporter)
	if !ok {
		t.Fatalf("expected PlaintextReporter, got %T", reporter)
	}
	if plain.builder.category != CategoryHost {
		t.Errorf("expected CategoryHost, got %v", plain.builder.category)
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

func TestWriteReportsSingle(t *testing.T) {
	aggs := testAggregates()
	var buf bytes.Buffer
	cfg := ReportConfig{
		Category:     CategoryHost,
		Metric:       MetricUptime,
		Limit:        20,
		OutputFormat: FormatPlaintext,
	}
	if err := WriteReports(&buf, aggs, cfg); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	if !strings.Contains(buf.String(), "host1") {
		t.Error("expected output to contain host1")
	}
}

func TestWriteReportsAll(t *testing.T) {
	aggs := testAggregates()
	var buf bytes.Buffer
	cfg := ReportConfig{
		Category:      CategoryHost,
		Metric:        MetricUptime,
		Limit:         20,
		OutputFormat:  FormatPlaintext,
		All:           true,
		IncludeKernel: true,
	}
	if err := WriteReports(&buf, aggs, cfg); err != nil {
		t.Fatalf("WriteReports: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Uptime") {
		t.Error("expected output to contain Uptime report")
	}
	if !strings.Contains(out, "Boots") {
		t.Error("expected output to contain Boots report")
	}
}

func TestWriteReportsInvalidMetricForCategory(t *testing.T) {
	aggs := testAggregates()
	var buf bytes.Buffer
	cfg := ReportConfig{
		Category:     CategoryKernel,
		Metric:       MetricDowntime,
		Limit:        20,
		OutputFormat: FormatPlaintext,
	}
	err := WriteReports(&buf, aggs, cfg)
	if err == nil {
		t.Fatal("expected error for Downtime on Kernel category")
	}
	if !strings.Contains(err.Error(), "only supports") {
		t.Errorf("unexpected error: %v", err)
	}
}

func testAggregates() *Aggregates {
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
	kernel := NewAggregate("Linux 5.10")
	kernel.Uptime = 86400000
	kernel.Boots = 10
	aggs.Kernel["Linux 5.10"] = kernel
	return aggs
}

func TestParseReportQuery(t *testing.T) {
	q := url.Values{}
	cfg, err := ParseReportQuery(q)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Category != CategoryHost || cfg.Metric != MetricUptime || cfg.Limit != 20 {
		t.Fatalf("defaults: %+v", cfg)
	}
	if cfg.OutputFormat != FormatPlaintext || cfg.All || cfg.IncludeKernel || cfg.StatsOrder != "" {
		t.Fatalf("defaults: %+v", cfg)
	}
	q.Set("category", "Kernel")
	q.Set("metric", "Boots")
	q.Set("limit", "5")
	q.Set("output-format", "Markdown")
	q.Set("all", "true")
	q.Set("include-kernel", "1")
	q.Set("stats-order", "Host:Uptime")
	cfg, err = ParseReportQuery(q)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Category != CategoryKernel || cfg.Metric != MetricBoots || cfg.Limit != 5 {
		t.Fatalf("got %+v", cfg)
	}
	if cfg.OutputFormat != FormatMarkdown || !cfg.All || !cfg.IncludeKernel || cfg.StatsOrder != "Host:Uptime" {
		t.Fatalf("got %+v", cfg)
	}
	_, err = ParseReportQuery(url.Values{"category": []string{"nope"}})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseReportQuery(url.Values{"limit": []string{"x"}})
	if err == nil {
		t.Fatal("expected error")
	}
	_, err = ParseReportQuery(url.Values{"all": []string{"maybe"}})
	if err == nil {
		t.Fatal("expected error")
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
