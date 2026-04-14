package goprecords

import (
	"testing"
	"time"
)

func TestNewAggregate(t *testing.T) {
	agg := NewAggregate("testhost")
	if agg.Name != "testhost" {
		t.Errorf("got %q, want %q", agg.Name, "testhost")
	}
	if agg.Uptime != 0 || agg.Boots != 0 {
		t.Errorf("expected zero values")
	}
}

func TestNewHostAggregate(t *testing.T) {
	hagg := NewHostAggregate("testhost", "Linux 5.10.0")
	if hagg.Name != "testhost" {
		t.Errorf("got %q, want %q", hagg.Name, "testhost")
	}
	if hagg.LastKernel != "Linux 5.10.0" {
		t.Errorf("got %q, want %q", hagg.LastKernel, "Linux 5.10.0")
	}
}

func TestAddRecord(t *testing.T) {
	agg := NewAggregate("host1")
	agg.AddRecord(1000, 2000)
	if agg.Uptime != 1000 {
		t.Errorf("got %d, want 1000", agg.Uptime)
	}
	if agg.Boots != 1 {
		t.Errorf("got %d, want 1", agg.Boots)
	}
	if agg.FirstBoot != 2000 {
		t.Errorf("got %d, want 2000", agg.FirstBoot)
	}
	if agg.LastSeen != 3000 {
		t.Errorf("got %d, want 3000", agg.LastSeen)
	}
}

func TestAddRecordMultiple(t *testing.T) {
	agg := NewAggregate("host1")
	agg.AddRecord(1000, 2000)
	agg.AddRecord(500, 1000)
	agg.AddRecord(1500, 3000)

	if agg.Uptime != 3000 {
		t.Errorf("got %d, want 3000", agg.Uptime)
	}
	if agg.Boots != 3 {
		t.Errorf("got %d, want 3", agg.Boots)
	}
	if agg.FirstBoot != 1000 {
		t.Errorf("got %d, want 1000", agg.FirstBoot)
	}
	if agg.LastSeen != 4500 {
		t.Errorf("got %d, want 4500", agg.LastSeen)
	}
}

func TestIsActive(t *testing.T) {
	agg := NewAggregate("host1")
	now := uint64(time.Now().Unix())
	agg.LastSeen = now // Very recent

	if !agg.IsActive(90) {
		t.Error("expected IsActive(90) to be true for recent LastSeen")
	}

	agg.LastSeen = now - (100 * 24 * 3600) // 100 days ago
	if agg.IsActive(90) {
		t.Error("expected IsActive(90) to be false for LastSeen 100 days ago")
	}
}

func TestMetaScore(t *testing.T) {
	agg := NewAggregate("host1")
	agg.Uptime = 86400000 // Large uptime
	agg.Boots = 100
	agg.LastSeen = uint64(time.Now().Unix())
	agg.FirstBoot = agg.LastSeen - 1000000

	score := agg.MetaScore()
	if score == 0 {
		t.Error("expected non-zero MetaScore")
	}
}

func TestHostAggregateLifespan(t *testing.T) {
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.FirstBoot = 1000
	hagg.LastSeen = 5000

	lifespan := hagg.Lifespan()
	if lifespan != 4000 {
		t.Errorf("got %d, want 4000", lifespan)
	}
}

func TestHostAggregateDowntime(t *testing.T) {
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.FirstBoot = 1000
	hagg.LastSeen = 5000
	hagg.Uptime = 3000

	downtime := hagg.Downtime()
	if downtime != 1000 {
		t.Errorf("got %d, want 1000", downtime)
	}
}

func TestHostAggregateLifespanUnderflow(t *testing.T) {
	hagg := NewHostAggregate("host1", "Linux 5.10")
	hagg.FirstBoot = 5000
	hagg.LastSeen = 1000

	if got := hagg.Lifespan(); got != 0 {
		t.Errorf("Lifespan() = %d, want 0 when LastSeen < FirstBoot", got)
	}
}

func TestHostAggregateDowntimeUnderflow(t *testing.T) {
	tests := []struct {
		name      string
		firstBoot uint64
		lastSeen  uint64
		uptime    uint64
	}{
		{"uptime equals lifespan", 1000, 5000, 4000},
		{"uptime exceeds lifespan", 1000, 5000, 5000},
		{"uptime exceeds short span", 0, 100, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hagg := NewHostAggregate("host1", "Linux 5.10")
			hagg.FirstBoot = tt.firstBoot
			hagg.LastSeen = tt.lastSeen
			hagg.Uptime = tt.uptime
			if got := hagg.Downtime(); got != 0 {
				t.Errorf("Downtime() = %d, want 0", got)
			}
		})
	}
}

func TestEpochHumanDuration(t *testing.T) {
	// Unix epoch + 1 year + 2 months + 3 days
	epoch := Epoch(31536000 + (60 * 24 * 3600) + (3 * 24 * 3600))
	duration := epoch.HumanDuration()

	if duration == "" {
		t.Error("expected non-empty duration string")
	}
	// Should contain years, months, days
	if !contains(duration, "years") || !contains(duration, "months") || !contains(duration, "days") {
		t.Errorf("unexpected duration format: %s", duration)
	}
}

func TestEpochNewerThan(t *testing.T) {
	now := uint64(time.Now().Unix())

	// Recent epoch
	recent := Epoch(now - 10*24*3600) // 10 days ago
	if !recent.NewerThan(20) {
		t.Error("expected recent epoch to be newer than 20 days")
	}

	// Old epoch
	old := Epoch(now - 100*24*3600) // 100 days ago
	if old.NewerThan(90) {
		t.Error("expected old epoch to not be newer than 90 days")
	}
}

func TestCategoryString(t *testing.T) {
	tests := []struct {
		cat  Category
		want string
	}{
		{CategoryHost, "Host"},
		{CategoryKernel, "Kernel"},
		{CategoryKernelMajor, "KernelMajor"},
		{CategoryKernelName, "KernelName"},
		{Category(999), "?"},
	}

	for _, tt := range tests {
		got := tt.cat.String()
		if got != tt.want {
			t.Errorf("Category(%d).String() = %q, want %q", tt.cat, got, tt.want)
		}
	}
}

func TestMetricString(t *testing.T) {
	tests := []struct {
		met  Metric
		want string
	}{
		{MetricBoots, "Boots"},
		{MetricUptime, "Uptime"},
		{MetricScore, "Score"},
		{MetricDowntime, "Downtime"},
		{MetricLifespan, "Lifespan"},
		{Metric(999), "?"},
	}

	for _, tt := range tests {
		got := tt.met.String()
		if got != tt.want {
			t.Errorf("Metric(%d).String() = %q, want %q", tt.met, got, tt.want)
		}
	}
}

func TestOutputFormatString(t *testing.T) {
	tests := []struct {
		fmt  OutputFormat
		want string
	}{
		{FormatPlaintext, "Plaintext"},
		{FormatMarkdown, "Markdown"},
		{FormatGemtext, "Gemtext"},
		{FormatHTML, "HTML"},
		{OutputFormat(999), "?"},
	}

	for _, tt := range tests {
		got := tt.fmt.String()
		if got != tt.want {
			t.Errorf("OutputFormat(%d).String() = %q, want %q", tt.fmt, got, tt.want)
		}
	}
}

func TestMetricDescription(t *testing.T) {
	tests := []struct {
		metric   Metric
		contains string
	}{
		{MetricBoots, "boots"},
		{MetricUptime, "uptime"},
		{MetricScore, "Score"},
		{MetricDowntime, "downtime"},
		{MetricLifespan, "uptime"},
	}

	for _, tt := range tests {
		desc := MetricDescription(tt.metric)
		if desc == "" {
			t.Errorf("MetricDescription(%v) returned empty string", tt.metric)
		}
	}
}

func TestWordWrap(t *testing.T) {
	tests := []struct {
		text  string
		limit int
		name  string
	}{
		{"short text", 100, "short text no wrap"},
		{"this is a very long text that should be wrapped at some point because it exceeds the limit", 30, "long text wrap"},
		{"", 50, "empty string"},
	}

	for _, tt := range tests {
		result := wordWrap(tt.text, tt.limit)
		lines := 0
		for _, line := range result {
			if line == '\n' {
				lines++
			}
		}

		// Just verify it doesn't crash and returns something reasonable
		if len(result) == 0 && len(tt.text) > 0 {
			t.Errorf("wordWrap(%q, %d): returned empty for non-empty input", tt.text, tt.limit)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	result := formatDuration(86400)
	if result == "" {
		t.Error("formatDuration returned empty string")
	}
}

func TestFormatInt(t *testing.T) {
	tests := []struct {
		n    uint64
		want string
	}{
		{0, "0"},
		{123, "123"},
		{9999999, "9999999"},
	}

	for _, tt := range tests {
		got := formatInt(tt.n)
		if got != tt.want {
			t.Errorf("formatInt(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
