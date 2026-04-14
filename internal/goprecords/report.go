package goprecords

import (
	"fmt"
	"io"
	"sort"
)

// WriteReports renders reports to w based on the given config.
func WriteReports(w io.Writer, aggregates *Aggregates, cfg ReportConfig) error {
	if !cfg.All {
		if cfg.Category != CategoryHost && (cfg.Metric == MetricDowntime || cfg.Metric == MetricLifespan) {
			return fmt.Errorf("Category %s only supports: Boots, Uptime, Score", cfg.Category)
		}
		if cfg.Category == CategoryHost {
			io.WriteString(w, NewHostReporter(aggregates, cfg.Limit, cfg.Metric, cfg.OutputFormat, 1).Report())
		} else {
			io.WriteString(w, NewReporter(aggregates, cfg.Category, cfg.Limit, cfg.Metric, cfg.OutputFormat, 1).Report())
		}
		return nil
	}
	order, err := StatsOrderList(cfg.StatsOrder)
	if err != nil {
		return err
	}
	headerIndent := uint(2)
	for _, pair := range order {
		c, m := pair.Category, pair.Metric
		if !cfg.IncludeKernel && c == CategoryKernel {
			continue
		}
		if c != CategoryHost && (m == MetricDowntime || m == MetricLifespan) {
			continue
		}
		if c == CategoryHost {
			io.WriteString(w, NewHostReporter(aggregates, cfg.Limit, m, cfg.OutputFormat, headerIndent).Report())
		} else {
			io.WriteString(w, NewReporter(aggregates, c, cfg.Limit, m, cfg.OutputFormat, headerIndent).Report())
		}
		io.WriteString(w, "\n")
	}
	return nil
}

type Reporter interface {
	Report() string
}

type reportBuilder struct {
	aggregates   *Aggregates
	limit        uint
	category     Category
	metric       Metric
	headerIndent uint
}

type PlaintextReporter struct {
	builder reportBuilder
}

type MarkdownReporter struct {
	builder reportBuilder
}

type GemtextReporter struct {
	builder reportBuilder
}

type HTMLReporter struct {
	builder reportBuilder
}

func NewReporter(aggregates *Aggregates, category Category, limit uint, metric Metric, outputFormat OutputFormat, headerIndent uint) Reporter {
	builder := reportBuilder{
		aggregates:   aggregates,
		limit:        limit,
		category:     category,
		metric:       metric,
		headerIndent: headerIndent,
	}
	switch outputFormat {
	case FormatMarkdown:
		return &MarkdownReporter{builder: builder}
	case FormatGemtext:
		return &GemtextReporter{builder: builder}
	case FormatHTML:
		return &HTMLReporter{builder: builder}
	default:
		return &PlaintextReporter{builder: builder}
	}
}

func NewHostReporter(aggregates *Aggregates, limit uint, metric Metric, outputFormat OutputFormat, headerIndent uint) Reporter {
	return NewReporter(aggregates, CategoryHost, limit, metric, outputFormat, headerIndent)
}

func (r *PlaintextReporter) Report() string {
	return r.builder.Report(FormatPlaintext)
}

func (r *MarkdownReporter) Report() string {
	return r.builder.Report(FormatMarkdown)
}

func (r *GemtextReporter) Report() string {
	return r.builder.Report(FormatGemtext)
}

func (r *HTMLReporter) Report() string {
	return r.builder.Report(FormatHTML)
}

func (r reportBuilder) Report(outputFormat OutputFormat) string {
	var rows []tableRow
	var hasLastKernel bool
	if r.category == CategoryHost {
		rows, hasLastKernel = r.buildHostTable()
	} else {
		rows, hasLastKernel = r.buildCategoryTable()
	}
	if len(rows) == 0 {
		return ""
	}
	if outputFormat == FormatHTML {
		return r.formatReportHTML(rows, hasLastKernel)
	}
	return r.formatReport(rows, hasLastKernel, outputFormat)
}

func (r reportBuilder) buildHostTable() ([]tableRow, bool) {
	type keyVal struct {
		agg *HostAggregate
		key uint64
	}
	var list []keyVal
	for _, h := range r.aggregates.Host {
		var k uint64
		switch r.metric {
		case MetricUptime:
			k = h.Uptime
		case MetricBoots:
			k = h.Boots
		case MetricScore:
			k = h.MetaScore()
		case MetricDowntime:
			k = h.Downtime()
		case MetricLifespan:
			k = h.Lifespan()
		default:
			k = h.Uptime
		}
		list = append(list, keyVal{h, k})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].key > list[j].key })
	var rows []tableRow
	for i, kv := range list {
		if uint(i) >= r.limit {
			break
		}
		h := kv.agg
		active := " "
		if h.IsActive(90) {
			active = "*"
		}
		rows = append(rows, tableRow{
			Pos:        fmt.Sprintf("%d.", i+1),
			Name:       active + h.Name,
			Value:      r.humanStrHost(h),
			LastKernel: h.LastKernel,
		})
	}
	return rows, true
}

func (r reportBuilder) buildCategoryTable() ([]tableRow, bool) {
	m := r.aggregates.Kernel
	switch r.category {
	case CategoryKernelMajor:
		m = r.aggregates.KernelMajor
	case CategoryKernelName:
		m = r.aggregates.KernelName
	}
	type keyVal struct {
		agg *Aggregate
		key uint64
	}
	var list []keyVal
	for _, a := range m {
		var k uint64
		switch r.metric {
		case MetricUptime:
			k = a.Uptime
		case MetricBoots:
			k = a.Boots
		case MetricScore:
			k = a.MetaScore()
		default:
			k = a.Uptime
		}
		list = append(list, keyVal{agg: a, key: k})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].key > list[j].key })
	var rows []tableRow
	for i, kv := range list {
		if uint(i) >= r.limit {
			break
		}
		a := kv.agg
		active := " "
		if a.IsActive(90) {
			active = "*"
		}
		rows = append(rows, tableRow{
			Pos:   fmt.Sprintf("%d.", i+1),
			Name:  active + a.Name,
			Value: r.humanStrAgg(a),
		})
	}
	return rows, false
}

func (r reportBuilder) humanStrHost(h *HostAggregate) string {
	switch r.metric {
	case MetricUptime:
		return formatDuration(h.Uptime)
	case MetricBoots:
		return formatInt(h.Boots)
	case MetricScore:
		return formatInt(h.MetaScore())
	case MetricDowntime:
		return formatDuration(h.Downtime())
	case MetricLifespan:
		return formatDuration(h.Lifespan())
	default:
		return formatDuration(h.Uptime)
	}
}

func (r reportBuilder) humanStrAgg(a *Aggregate) string {
	switch r.metric {
	case MetricUptime:
		return formatDuration(a.Uptime)
	case MetricBoots:
		return formatInt(a.Boots)
	case MetricScore:
		return formatInt(a.MetaScore())
	default:
		return formatDuration(a.Uptime)
	}
}
