package recordline

import (
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in   string
		want Fields
		ok   bool
	}{
		{
			"12345:1700000000:Linux 6.5.0-generic",
			Fields{
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
			Fields{
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
			Fields{
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
			Fields{
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
			Fields{
				Uptime:      0,
				BootTime:    0,
				OS:          "Linux 6.5.0",
				KernelName:  "Linux",
				KernelMajor: "Linux 6...",
			},
			true,
		},
		{"", Fields{}, false},
		{"   ", Fields{}, false},
		{"only:two", Fields{}, false},
		{"no-colons-at-all", Fields{}, false},
	}
	for _, tt := range tests {
		got, ok := Parse(tt.in)
		if ok != tt.ok {
			t.Errorf("Parse(%q) ok=%v, want %v", tt.in, ok, tt.ok)
			continue
		}
		if tt.ok && got != tt.want {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}
