package goprecords

import (
	"fmt"
	"io"
	"sort"
)

type metricExtractor struct {
	hostSortKey func(*HostAggregate) uint64
	aggSortKey  func(*Aggregate) uint64
	hostHuman   func(*HostAggregate) string
	aggHuman    func(*Aggregate) string
}

var uptimeMetricExtractor = metricExtractor{
	hostSortKey: func(h *HostAggregate) uint64 { return h.Uptime },
	aggSortKey:  func(a *Aggregate) uint64 { return a.Uptime },
	hostHuman:   func(h *HostAggregate) string { return formatDuration(h.Uptime) },
	aggHuman:    func(a *Aggregate) string { return formatDuration(a.Uptime) },
}

var metricExtractors = map[Metric]metricExtractor{
	MetricBoots: {
		hostSortKey: func(h *HostAggregate) uint64 { return h.Boots },
		aggSortKey:  func(a *Aggregate) uint64 { return a.Boots },
		hostHuman:   func(h *HostAggregate) string { return formatInt(h.Boots) },
		aggHuman:    func(a *Aggregate) string { return formatInt(a.Boots) },
	},
	MetricUptime: uptimeMetricExtractor,
	MetricScore: {
		hostSortKey: func(h *HostAggregate) uint64 { return h.MetaScore() },
		aggSortKey:  func(a *Aggregate) uint64 { return a.MetaScore() },
		hostHuman:   func(h *HostAggregate) string { return formatInt(h.MetaScore()) },
		aggHuman:    func(a *Aggregate) string { return formatInt(a.MetaScore()) },
	},
	MetricDowntime: {
		hostSortKey: func(h *HostAggregate) uint64 { return h.Downtime() },
		aggSortKey:  func(a *Aggregate) uint64 { return a.Uptime },
		hostHuman:   func(h *HostAggregate) string { return formatDuration(h.Downtime()) },
		aggHuman:    func(a *Aggregate) string { return formatDuration(a.Uptime) },
	},
	MetricLifespan: {
		hostSortKey: func(h *HostAggregate) uint64 { return h.Lifespan() },
		aggSortKey:  func(a *Aggregate) uint64 { return a.Uptime },
		hostHuman:   func(h *HostAggregate) string { return formatDuration(h.Lifespan()) },
		aggHuman:    func(a *Aggregate) string { return formatDuration(a.Uptime) },
	},
}

func extractorFor(m Metric) metricExtractor {
	if e, ok := metricExtractors[m]; ok {
		return e
	}
	return uptimeMetricExtractor
}

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
	ex := extractorFor(r.metric)
	for _, h := range r.aggregates.Host {
		k := ex.hostSortKey(h)
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
	ex := extractorFor(r.metric)
	for _, a := range m {
		k := ex.aggSortKey(a)
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
	return extractorFor(r.metric).hostHuman(h)
}

func (r reportBuilder) humanStrAgg(a *Aggregate) string {
	return extractorFor(r.metric).aggHuman(a)
}
