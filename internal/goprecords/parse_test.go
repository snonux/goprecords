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

func TestParseRecordLine(t *testing.T) {
	tests := []struct {
		in   string
		want recordLine
		ok   bool
	}{
		{
			"12345:1700000000:Linux 6.5.0-generic",
			recordLine{
				Uptime:      12345,
				BootTime:    1700000000,
				OS:          "Linux 6.5.0-generic",
				KernelName:  "Linux",
				KernelMajor: "Linux 6...",
			},
			true,
		},
		{
			"  99:100:FreeBSD 14.0-RELEASE  ",
			recordLine{
				Uptime:      99,
				BootTime:    100,
				OS:          "FreeBSD 14.0-RELEASE",
				KernelName:  "FreeBSD",
				KernelMajor: "FreeBSD 14...",
			},
			true,
		},
		{
			"500:200:SingleToken",
			recordLine{
				Uptime:      500,
				BootTime:    200,
				OS:          "SingleToken",
				KernelName:  "SingleToken",
				KernelMajor: "SingleToken SingleToken...",
			},
			true,
		},
		{
			"100:200:Linux 6.5.0:extra",
			recordLine{
				Uptime:      100,
				BootTime:    200,
				OS:          "Linux 6.5.0:extra",
				KernelName:  "Linux",
				KernelMajor: "Linux 6...",
			},
			true,
		},
		{
			"abc:def:Linux 6.5.0",
			recordLine{
				Uptime:      0,
				BootTime:    0,
				OS:          "Linux 6.5.0",
				KernelName:  "Linux",
				KernelMajor: "Linux 6...",
			},
			true,
		},
		{"", recordLine{}, false},
		{"   ", recordLine{}, false},
		{"only:two", recordLine{}, false},
		{"no-colons-at-all", recordLine{}, false},
	}
	for _, tt := range tests {
		got, ok := parseRecordLine(tt.in)
		if ok != tt.ok {
			t.Errorf("parseRecordLine(%q) ok=%v, want %v", tt.in, ok, tt.ok)
			continue
		}
		if tt.ok && got != tt.want {
			t.Errorf("parseRecordLine(%q) = %+v, want %+v", tt.in, got, tt.want)
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
