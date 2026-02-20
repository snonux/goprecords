package goprecords

import (
	"testing"
)

func TestParseStatsOrder(t *testing.T) {
	tests := []struct {
		in    string
		want  []CategoryMetric
		valid bool
	}{
		{
			in:    "Host:Uptime,Host:Boots",
			want:  []CategoryMetric{{CategoryHost, MetricUptime}, {CategoryHost, MetricBoots}},
			valid: true,
		},
		{
			in:    "Host:Uptime",
			want:  []CategoryMetric{{CategoryHost, MetricUptime}},
			valid: true,
		},
		{in: "Host", valid: false},
		{in: "Bad:Uptime", valid: false},
		{in: "Kernel:Downtime", valid: false},
		{in: "Host:Nope", valid: false},
		{in: "", valid: false},
		{in: "  ,  ", valid: false},
	}
	for _, tt := range tests {
		got, err := ParseStatsOrder(tt.in)
		valid := err == nil
		if valid != tt.valid {
			t.Errorf("ParseStatsOrder(%q) err=%v; valid=%v want %v", tt.in, err, valid, tt.valid)
			continue
		}
		if !tt.valid {
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("ParseStatsOrder(%q) len=%d want %d", tt.in, len(got), len(tt.want))
			continue
		}
		for i := range got {
			if got[i].Category != tt.want[i].Category || got[i].Metric != tt.want[i].Metric {
				t.Errorf("ParseStatsOrder(%q)[%d] = %v; want %v", tt.in, i, got[i], tt.want[i])
			}
		}
	}
}

func TestStatsOrderList(t *testing.T) {
	// Empty string returns default order (all category×metric pairs).
	got, err := StatsOrderList("")
	if err != nil {
		t.Fatalf("StatsOrderList(\"\"): %v", err)
	}
	if len(got) == 0 {
		t.Error("StatsOrderList(\"\"): got empty order")
	}
	// Custom order: Host:Uptime first, then rest of default.
	got, err = StatsOrderList("Host:Uptime")
	if err != nil {
		t.Fatalf("StatsOrderList(\"Host:Uptime\"): %v", err)
	}
	if len(got) == 0 {
		t.Fatal("StatsOrderList(\"Host:Uptime\"): got empty")
	}
	if got[0].Category != CategoryHost || got[0].Metric != MetricUptime {
		t.Errorf("StatsOrderList(\"Host:Uptime\")[0] = %v; want Host:Uptime", got[0])
	}
}
