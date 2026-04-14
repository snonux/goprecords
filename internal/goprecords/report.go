package goprecords

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// ReportConfig holds parsed report configuration.
type ReportConfig struct {
	Category      Category
	Metric        Metric
	Limit         uint
	OutputFormat  OutputFormat
	All           bool
	IncludeKernel bool
	StatsOrder    string
}

// ReportFlags holds flag pointers registered on a FlagSet.
type ReportFlags struct {
	category      *string
	metric        *string
	limit         *uint
	outputFormat  *string
	all           *bool
	includeKernel *bool
	statsOrder    *string
}

// RegisterReportFlags registers common report flags on the given FlagSet.
func RegisterReportFlags(fs *flag.FlagSet) *ReportFlags {
	return &ReportFlags{
		category:      fs.String("category", "Host", "Category: Host, Kernel, KernelMajor, KernelName"),
		metric:        fs.String("metric", "Uptime", "Metric: Boots, Uptime, Score, Downtime, Lifespan"),
		limit:         fs.Uint("limit", 20, "Limit output to num of entries"),
		outputFormat:  fs.String("output-format", "Plaintext", "Output format: Plaintext, Markdown, Gemtext"),
		all:           fs.Bool("all", false, "Generate all possible stats but Kernel"),
		includeKernel: fs.Bool("include-kernel", false, "Also include Kernel when using -all"),
		statsOrder:    fs.String("stats-order", "", "Comma-separated Category:Metric order for -all"),
	}
}

// Parse converts flag values into a ReportConfig.
func (rf *ReportFlags) Parse() (ReportConfig, error) {
	cat, err := ParseCategory(*rf.category)
	if err != nil {
		return ReportConfig{}, err
	}
	met, err := ParseMetric(*rf.metric)
	if err != nil {
		return ReportConfig{}, err
	}
	outFmt, err := ParseOutputFormat(*rf.outputFormat)
	if err != nil {
		return ReportConfig{}, err
	}
	return ReportConfig{
		Category:      cat,
		Metric:        met,
		Limit:         *rf.limit,
		OutputFormat:  outFmt,
		All:           *rf.all,
		IncludeKernel: *rf.includeKernel,
		StatsOrder:    *rf.statsOrder,
	}, nil
}

// ParseReportQuery builds a ReportConfig from URL query parameters using the
// same names and defaults as RegisterReportFlags (category, metric, limit,
// output-format, all, include-kernel, stats-order).
func ParseReportQuery(q url.Values) (ReportConfig, error) {
	catStr := q.Get("category")
	if catStr == "" {
		catStr = "Host"
	}
	cat, err := ParseCategory(catStr)
	if err != nil {
		return ReportConfig{}, err
	}
	metStr := q.Get("metric")
	if metStr == "" {
		metStr = "Uptime"
	}
	met, err := ParseMetric(metStr)
	if err != nil {
		return ReportConfig{}, err
	}
	limit := uint(20)
	if ls := q.Get("limit"); ls != "" {
		v, err := strconv.ParseUint(ls, 10, 32)
		if err != nil {
			return ReportConfig{}, fmt.Errorf("invalid limit %q", ls)
		}
		limit = uint(v)
	}
	outStr := q.Get("output-format")
	if outStr == "" {
		outStr = "Plaintext"
	}
	outFmt, err := ParseOutputFormat(outStr)
	if err != nil {
		return ReportConfig{}, err
	}
	all := false
	if v := q.Get("all"); v != "" {
		all, err = parseQueryBool(v)
		if err != nil {
			return ReportConfig{}, err
		}
	}
	includeKernel := false
	if v := q.Get("include-kernel"); v != "" {
		includeKernel, err = parseQueryBool(v)
		if err != nil {
			return ReportConfig{}, err
		}
	}
	return ReportConfig{
		Category:      cat,
		Metric:        met,
		Limit:         limit,
		OutputFormat:  outFmt,
		All:           all,
		IncludeKernel: includeKernel,
		StatsOrder:    q.Get("stats-order"),
	}, nil
}

func parseQueryBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no", "":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", s)
	}
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

func (r reportBuilder) formatReport(rows []tableRow, hasLastKernel bool, outputFormat OutputFormat) string {
	cW, nW, vW, lkW := r.reportWidths(rows, hasLastKernel)
	border := r.buildBorder(cW, nW, vW, lkW, hasLastKernel)
	header := r.buildReportHeader(cW, nW, vW, lkW, hasLastKernel, border, outputFormat)
	fmtStr := r.buildFormatStr(cW, nW, vW, lkW, hasLastKernel)
	body := r.buildReportBody(rows, fmtStr, hasLastKernel)
	out := header + body + border
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		out += "```\n"
	}
	return out
}

func (r reportBuilder) reportWidths(rows []tableRow, hasLastKernel bool) (countW, nameW, valueW, lastKernelW int) {
	countW = 3
	nameW = len(r.category.String())
	valueW = len(r.metric.String())
	if hasLastKernel {
		lastKernelW = len("Last Kernel")
	}
	for _, row := range rows {
		if len(row.Pos) > countW {
			countW = len(row.Pos)
		}
		if len(row.Name) > nameW {
			nameW = len(row.Name)
		}
		if len(row.Value) > valueW {
			valueW = len(row.Value)
		}
		if len(row.LastKernel) > lastKernelW {
			lastKernelW = len(row.LastKernel)
		}
	}
	return countW, nameW, valueW, lastKernelW
}

func (r reportBuilder) buildBorder(countW, nameW, valueW, lastKernelW int, hasLastKernel bool) string {
	parts := []string{
		"+" + strings.Repeat("-", 2+countW),
		"+" + strings.Repeat("-", 2+nameW),
		"+" + strings.Repeat("-", 2+valueW),
	}
	if hasLastKernel {
		parts = append(parts, "+"+strings.Repeat("-", 2+lastKernelW))
	}
	return strings.Join(parts, "") + "+\n"
}

func (r reportBuilder) buildReportHeader(countW, nameW, valueW, lastKernelW int, hasLastKernel bool, border string, outputFormat OutputFormat) string {
	var h string
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		h = strings.Repeat("#", int(r.headerIndent)) + " "
	}
	h += fmt.Sprintf("Top %d %s's by %s\n\n", r.limit, r.metric, r.category)
	desc := MetricDescription(r.metric)
	lineLimit := len(border)
	if outputFormat == FormatPlaintext && lineLimit > 0 && len(desc) > lineLimit-1 {
		desc = " " + wordWrap(desc, lineLimit-1)
	}
	h += desc + "\n\n"
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		h += "```\n"
	}
	h += border
	fmtStr := r.buildFormatStr(countW, nameW, valueW, lastKernelW, hasLastKernel)
	if hasLastKernel {
		h += fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String(), "Last Kernel")
	} else {
		h += fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String())
	}
	h += border
	return h
}

func (r reportBuilder) buildFormatStr(countW, nameW, valueW, lastKernelW int, hasLastKernel bool) string {
	if hasLastKernel {
		return fmt.Sprintf("| %%%ds | %%%ds | %%%ds | %%%ds |", countW, nameW, valueW, lastKernelW)
	}
	return fmt.Sprintf("| %%%ds | %%%ds | %%%ds |", countW, nameW, valueW)
}

func (r reportBuilder) buildReportBody(rows []tableRow, fmtStr string, hasLastKernel bool) string {
	var b strings.Builder
	for _, row := range rows {
		if hasLastKernel {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value, row.LastKernel))
		} else {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value))
		}
	}
	return b.String()
}
