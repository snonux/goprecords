package goprecords

import (
	"testing"
)

func TestParseCategory(t *testing.T) {
	tests := []struct {
		in   string
		want Category
		ok   bool
	}{
		{"Host", CategoryHost, true},
		{"Kernel", CategoryKernel, true},
		{"KernelMajor", CategoryKernelMajor, true},
		{"KernelName", CategoryKernelName, true},
		{"", 0, false},
		{"host", 0, false},
		{"Bad", 0, false},
	}
	for _, tt := range tests {
		got, err := ParseCategory(tt.in)
		ok := err == nil
		if ok != tt.ok || (tt.ok && got != tt.want) {
			t.Errorf("ParseCategory(%q) = %v, %v; want %v, ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func TestParseMetric(t *testing.T) {
	tests := []struct {
		in   string
		want Metric
		ok   bool
	}{
		{"Boots", MetricBoots, true},
		{"Uptime", MetricUptime, true},
		{"Score", MetricScore, true},
		{"Downtime", MetricDowntime, true},
		{"Lifespan", MetricLifespan, true},
		{"", 0, false},
		{"uptime", 0, false},
		{"Nope", 0, false},
	}
	for _, tt := range tests {
		got, err := ParseMetric(tt.in)
		ok := err == nil
		if ok != tt.ok || (tt.ok && got != tt.want) {
			t.Errorf("ParseMetric(%q) = %v, %v; want %v, ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}

func TestParseOutputFormat(t *testing.T) {
	tests := []struct {
		in   string
		want OutputFormat
		ok   bool
	}{
		{"Plaintext", FormatPlaintext, true},
		{"Markdown", FormatMarkdown, true},
		{"Gemtext", FormatGemtext, true},
		{"", 0, false},
		{"html", 0, false},
	}
	for _, tt := range tests {
		got, err := ParseOutputFormat(tt.in)
		ok := err == nil
		if ok != tt.ok || (tt.ok && got != tt.want) {
			t.Errorf("ParseOutputFormat(%q) = %v, %v; want %v, ok=%v", tt.in, got, err, tt.want, tt.ok)
		}
	}
}
