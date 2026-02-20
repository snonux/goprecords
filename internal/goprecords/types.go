package goprecords

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	// Day is seconds in 24 hours.
	Day   = 24 * 3600
	// Month is 30 days in seconds.
	Month = 30 * Day
)

// Category is the grouping for reports (Host, Kernel, etc.).
type Category int

const (
	CategoryHost Category = iota
	CategoryKernel
	CategoryKernelMajor
	CategoryKernelName
)

// Metric is the value to rank by (Boots, Uptime, etc.).
type Metric int

const (
	MetricBoots Metric = iota
	MetricUptime
	MetricScore
	MetricDowntime
	MetricLifespan
)

// OutputFormat is the report output format.
type OutputFormat int

const (
	FormatPlaintext OutputFormat = iota
	FormatMarkdown
	FormatGemtext
)

// Epoch is a Unix timestamp for duration/date formatting.
type Epoch uint64

// Aggregate holds per-entity stats (Host, Kernel, etc.).
type Aggregate struct {
	Name      string
	Uptime    uint64
	FirstBoot uint64
	LastSeen  uint64
	Boots     uint64
}

// NewAggregate constructs an Aggregate with the given name.
func NewAggregate(name string) *Aggregate {
	return &Aggregate{Name: name}
}

// HostAggregate adds last-kernel and lifespan/downtime for host reports.
type HostAggregate struct {
	Aggregate
	LastKernel string
}

// NewHostAggregate constructs a HostAggregate.
func NewHostAggregate(name, lastKernel string) *HostAggregate {
	return &HostAggregate{
		Aggregate:  Aggregate{Name: name},
		LastKernel: lastKernel,
	}
}

// tableRow is one row in the report table.
type tableRow struct {
	Pos        string
	Name       string
	Value      string
	LastKernel string
}

// String returns the category name.
func (c Category) String() string {
	switch c {
	case CategoryHost:
		return "Host"
	case CategoryKernel:
		return "Kernel"
	case CategoryKernelMajor:
		return "KernelMajor"
	case CategoryKernelName:
		return "KernelName"
	default:
		return "?"
	}
}

// String returns the metric name.
func (m Metric) String() string {
	switch m {
	case MetricBoots:
		return "Boots"
	case MetricUptime:
		return "Uptime"
	case MetricScore:
		return "Score"
	case MetricDowntime:
		return "Downtime"
	case MetricLifespan:
		return "Lifespan"
	default:
		return "?"
	}
}

// String returns the format name.
func (f OutputFormat) String() string {
	switch f {
	case FormatPlaintext:
		return "Plaintext"
	case FormatMarkdown:
		return "Markdown"
	case FormatGemtext:
		return "Gemtext"
	default:
		return "?"
	}
}

// HumanDuration returns a human-readable duration from epoch (e.g. "1 years, 2 months, 3 days").
func (e Epoch) HumanDuration() string {
	t := time.Unix(int64(e), 0).UTC()
	y := t.Year() - 1970
	m := int(t.Month())
	d := t.Day()
	return fmt.Sprintf("%d years, %d months, %d days", y, m, d)
}

// NewerThan reports whether the epoch is within the last limitDays days.
func (e Epoch) NewerThan(limitDays uint) bool {
	then := time.Unix(int64(e), 0)
	return time.Since(then) < time.Duration(limitDays)*24*time.Hour
}

// AddRecord adds one uptime record.
func (a *Aggregate) AddRecord(uptimeSec, bootTime uint64) {
	lastSeen := uptimeSec + bootTime
	a.Uptime += uptimeSec
	a.Boots++
	if a.FirstBoot == 0 || bootTime < a.FirstBoot {
		a.FirstBoot = bootTime
	}
	if lastSeen > a.LastSeen {
		a.LastSeen = lastSeen
	}
}

// IsActive reports whether the entity was seen within limitDays days.
func (a *Aggregate) IsActive(limitDays uint) bool {
	return Epoch(a.LastSeen).NewerThan(limitDays)
}

// MetaScore returns the computed score for this aggregate.
func (a *Aggregate) MetaScore() uint64 {
	activeBonus := uint64(0)
	if a.IsActive(90) {
		activeBonus = Month
	}
	return ((a.Uptime*2 + a.Boots*uint64(Day) + activeBonus) / 1000000)
}

// Lifespan returns last-seen minus first-boot.
func (h *HostAggregate) Lifespan() uint64 { return h.LastSeen - h.FirstBoot }

// Downtime returns lifespan minus uptime.
func (h *HostAggregate) Downtime() uint64 { return h.Lifespan() - h.Uptime }

// MetaScore returns the host-specific score (includes downtime component).
func (h *HostAggregate) MetaScore() uint64 {
	return uint64(h.Downtime()/2000000) + h.Aggregate.MetaScore()
}

// MetricDescription returns the description text for a metric.
func MetricDescription(m Metric) string {
	switch m {
	case MetricBoots:
		return "Boots is the total number of host boots over the entire lifespan."
	case MetricUptime:
		return "Uptime is the total uptime of a host over the entire lifespan."
	case MetricDowntime:
		return "Downtime is the total downtime of a host over the entire lifespan."
	case MetricLifespan:
		return "Lifespan is the total uptime + the total downtime of a host."
	case MetricScore:
		return "Score is calculated by combining all other metrics."
	default:
		return ""
	}
}

// ParseCategory parses a category string.
func ParseCategory(s string) (Category, error) {
	switch s {
	case "Host":
		return CategoryHost, nil
	case "Kernel":
		return CategoryKernel, nil
	case "KernelMajor":
		return CategoryKernelMajor, nil
	case "KernelName":
		return CategoryKernelName, nil
	default:
		return 0, fmt.Errorf("invalid category %q", s)
	}
}

// ParseMetric parses a metric string.
func ParseMetric(s string) (Metric, error) {
	switch s {
	case "Boots":
		return MetricBoots, nil
	case "Uptime":
		return MetricUptime, nil
	case "Score":
		return MetricScore, nil
	case "Downtime":
		return MetricDowntime, nil
	case "Lifespan":
		return MetricLifespan, nil
	default:
		return 0, fmt.Errorf("invalid metric %q", s)
	}
}

// ParseOutputFormat parses an output format string.
func ParseOutputFormat(s string) (OutputFormat, error) {
	switch s {
	case "Plaintext":
		return FormatPlaintext, nil
	case "Markdown":
		return FormatMarkdown, nil
	case "Gemtext":
		return FormatGemtext, nil
	default:
		return 0, fmt.Errorf("invalid output-format %q", s)
	}
}

func wordWrap(s string, lineLimit int) string {
	if lineLimit <= 0 || len(s) <= lineLimit {
		return s
	}
	var b strings.Builder
	chars := 0
	for _, word := range strings.Fields(s) {
		wordLen := len(word)
		needsSpace := chars > 0
		totalLen := wordLen
		if needsSpace {
			totalLen++
		}
		if chars+totalLen > lineLimit {
			if chars > 0 {
				b.WriteByte('\n')
			}
			b.WriteString(word)
			chars = wordLen
		} else {
			if needsSpace {
				b.WriteByte(' ')
			}
			b.WriteString(word)
			chars += totalLen
		}
	}
	return b.String()
}

func formatDuration(sec uint64) string {
	return Epoch(sec).HumanDuration()
}

func formatInt(n uint64) string {
	return strconv.FormatUint(n, 10)
}
