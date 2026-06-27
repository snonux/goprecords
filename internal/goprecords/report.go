package goprecords

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type metricExtractor struct {
	hostSortKey func(*HostAggregate) uint64
	aggSortKey  func(*Aggregate) uint64
	hostHuman   func(*HostAggregate) string
	aggHuman    func(*Aggregate) string
}

var uptimeMetricExtractor = metricExtractor{
	hostSortKey: func(h *HostAggregate) uint64 { return h.Stats.Uptime },
	aggSortKey:  func(a *Aggregate) uint64 { return a.Uptime },
	hostHuman:   func(h *HostAggregate) string { return formatDuration(h.Stats.Uptime) },
	aggHuman:    func(a *Aggregate) string { return formatDuration(a.Uptime) },
}

var metricExtractors = map[Metric]metricExtractor{
	MetricBoots: {
		hostSortKey: func(h *HostAggregate) uint64 { return h.Stats.Boots },
		aggSortKey:  func(a *Aggregate) uint64 { return a.Boots },
		hostHuman:   func(h *HostAggregate) string { return formatInt(h.Stats.Boots) },
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
	MetricLastUpdated: {
		hostSortKey: func(h *HostAggregate) uint64 {
			if h.LastUpdated.IsZero() {
				return 0
			}
			return uint64(h.LastUpdated.UTC().Unix())
		},
		aggSortKey: func(a *Aggregate) uint64 { return 0 },
		hostHuman: func(h *HostAggregate) string {
			if h.LastUpdated.IsZero() {
				return ""
			}
			return h.LastUpdated.UTC().Format("2006-01-02 15:04")
		},
		aggHuman: func(a *Aggregate) string { return "" },
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
		if cfg.Category != CategoryHost && (cfg.Metric == MetricDowntime || cfg.Metric == MetricLifespan || cfg.Metric == MetricLastUpdated) {
			return fmt.Errorf("Category %s only supports: Boots, Uptime, Score", cfg.Category)
		}
		s := reportForPair(aggregates, cfg.Category, cfg.Metric, cfg.Limit, 1, cfg.OutputFormat)
		return writeReportString(w, wrapIfHTML(s, cfg.OutputFormat))
	}
	order, err := StatsOrderList(cfg.StatsOrder)
	if err != nil {
		return err
	}
	headerIndent := uint(2)
	if cfg.OutputFormat == FormatHTML {
		var parts []string
		for _, pair := range order {
			if skipPair(cfg, pair.Category, pair.Metric) {
				continue
			}
			if s := reportForPair(aggregates, pair.Category, pair.Metric, cfg.Limit, headerIndent, cfg.OutputFormat); s != "" {
				parts = append(parts, s)
			}
		}
		return writeReportString(w, wrapHTMLDocument(strings.Join(parts, "\n")))
	}
	for _, pair := range order {
		if skipPair(cfg, pair.Category, pair.Metric) {
			continue
		}
		s := reportForPair(aggregates, pair.Category, pair.Metric, cfg.Limit, headerIndent, cfg.OutputFormat)
		if err := writeReportString(w, s); err != nil {
			return err
		}
		if err := writeReportString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

func reportForPair(aggregates *Aggregates, c Category, m Metric, limit, headerIndent uint, outputFormat OutputFormat) string {
	if c == CategoryHost {
		return NewHostReporter(aggregates, limit, m, outputFormat, headerIndent).Report()
	}
	return NewReporter(aggregates, c, limit, m, outputFormat, headerIndent).Report()
}

func skipPair(cfg ReportConfig, c Category, m Metric) bool {
	if !cfg.IncludeKernel && c == CategoryKernel {
		return true
	}
	if c != CategoryHost && (m == MetricDowntime || m == MetricLifespan || m == MetricLastUpdated) {
		return true
	}
	return false
}

func wrapIfHTML(s string, f OutputFormat) string {
	if f != FormatHTML {
		return s
	}
	return wrapHTMLDocument(s)
}

func wrapHTMLDocument(body string) string {
	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"UTF-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	b.WriteString("<title>goprecords uptime report</title>\n")
	b.WriteString(htmlStyle)
	b.WriteString("</head>\n<body>\n")
	b.WriteString(body)
	b.WriteString("</body>\n</html>\n")
	return b.String()
}

const htmlStyle = `<style>
        /* Compact, full-width layout */
        :root {
            --pad: 8px;
        }
        html, body {
            height: 100%;
        }
        body {
            font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
            line-height: 1.3;
            margin: 0;
            padding: var(--pad);
            background: #fff;
            color: #000;
        }
        /* Headings: smaller and tighter */
        h1, h2, h3 { margin: 0.5em 0 0.25em; font-weight: 600; }
        h1 { font-size: 1em; }
        h2 { font-size: 0.95em; }
        h3 { font-size: 0.9em; }
        /* Paragraphs and lists: minimal vertical rhythm */
        p { margin: 0.2em 0; }
        ul { margin: 0.3em 0; padding-left: 1.2em; }
        li { margin: 0.1em 0; }
        /* Code blocks and tables */
        pre {
            overflow-x: auto;
            white-space: pre;
            margin: 0.3em 0;
        }
        table {
            border-collapse: collapse;
            table-layout: auto; /* size columns by content */
            width: auto;        /* do not stretch to full width */
            max-width: 100%;
            margin: 0.5em 0;
            font-size: 0.95em;
            display: inline-table; /* keep as compact as content allows */
        }
        th, td {
            padding: 0.1em 0.3em;
            text-align: left;
            white-space: nowrap; /* avoid wide columns caused by wrapping */
        }
        /* Links */
        a { color: #06c; text-decoration: underline; }
        a:visited { color: #639; }
        /* Rules */
        hr { border: none; border-top: 1px solid #ccc; margin: 0.5em 0; }
    </style>
`

func writeReportString(w io.Writer, s string) error {
	_, err := io.WriteString(w, s)
	return err
}

// Reporter renders aggregate uptime statistics as a string for one category and metric.
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

// PlaintextReporter formats a report as plain text.
type PlaintextReporter struct {
	builder reportBuilder
}

// MarkdownReporter formats a report as Markdown.
type MarkdownReporter struct {
	builder reportBuilder
}

// GemtextReporter formats a report as Gemtext.
type GemtextReporter struct {
	builder reportBuilder
}

// HTMLReporter formats a report as HTML.
type HTMLReporter struct {
	builder reportBuilder
}

// NewReporter returns a Reporter for category and metric, using outputFormat and
// limiting to the top limit entities. headerIndent controls heading depth for
// Markdown, Gemtext, and HTML.
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

// NewHostReporter returns a Reporter for the Host category with the given metric
// and output settings. It is equivalent to NewReporter with CategoryHost.
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
	var hasLastKernel, hasLastUpdated bool
	if r.category == CategoryHost {
		rows, hasLastKernel, hasLastUpdated = r.buildHostTable()
	} else {
		rows, hasLastKernel = r.buildCategoryTable()
	}
	if len(rows) == 0 {
		return ""
	}
	if outputFormat == FormatHTML {
		return r.formatReportHTML(rows, hasLastKernel, hasLastUpdated)
	}
	return r.formatReport(rows, hasLastKernel, hasLastUpdated, outputFormat)
}

func (r reportBuilder) buildHostTable() ([]tableRow, bool, bool) {
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
	var hasLastUpdated bool
	for i, kv := range list {
		if uint(i) >= r.limit {
			break
		}
		h := kv.agg
		active := " "
		if h.IsActive(90) {
			active = "*"
		}
		lastUpdated := ""
		if !h.LastUpdated.IsZero() {
			lastUpdated = h.LastUpdated.UTC().Format("2006-01-02 15:04")
			hasLastUpdated = true
		}
		rows = append(rows, tableRow{
			Pos:         fmt.Sprintf("%d.", i+1),
			Name:        active + h.Stats.Name,
			Value:       r.humanStrHost(h),
			LastKernel:  h.LastKernel,
			LastUpdated: lastUpdated,
		})
	}
	if r.metric == MetricLastUpdated {
		hasLastUpdated = false
	}
	return rows, true, hasLastUpdated
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
