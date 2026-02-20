package goprecords

import (
	"fmt"
	"sort"
	"strings"
)

// Reporter builds a single report (category + metric + format).
type Reporter struct {
	aggregates   *Aggregates
	limit        uint
	category     Category
	metric       Metric
	outputFormat OutputFormat
	headerIndent uint
}

// NewReporter returns a Reporter for the given category and metric.
func NewReporter(aggregates *Aggregates, category Category, limit uint, metric Metric, outputFormat OutputFormat, headerIndent uint) *Reporter {
	return &Reporter{
		aggregates:   aggregates,
		limit:        limit,
		category:     category,
		metric:       metric,
		outputFormat: outputFormat,
		headerIndent: headerIndent,
	}
}

// NewHostReporter returns a Reporter for Host category.
func NewHostReporter(aggregates *Aggregates, limit uint, metric Metric, outputFormat OutputFormat, headerIndent uint) *Reporter {
	return NewReporter(aggregates, CategoryHost, limit, metric, outputFormat, headerIndent)
}

// Report returns the formatted report string.
func (r Reporter) Report() string {
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
	return r.formatReport(rows, hasLastKernel)
}

func (r Reporter) buildHostTable() ([]tableRow, bool) {
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

func (r Reporter) buildCategoryTable() ([]tableRow, bool) {
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

func (r Reporter) humanStrHost(h *HostAggregate) string {
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

func (r Reporter) humanStrAgg(a *Aggregate) string {
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

func (r Reporter) formatReport(rows []tableRow, hasLastKernel bool) string {
	cW, nW, vW, lkW := r.reportWidths(rows, hasLastKernel)
	border := r.buildBorder(cW, nW, vW, lkW, hasLastKernel)
	header := r.buildReportHeader(cW, nW, vW, lkW, hasLastKernel, border)
	fmtStr := r.buildFormatStr(cW, nW, vW, lkW, hasLastKernel)
	body := r.buildReportBody(rows, fmtStr, hasLastKernel)
	out := header + body + border
	if r.outputFormat == FormatMarkdown || r.outputFormat == FormatGemtext {
		out += "```\n"
	}
	return out
}

func (r Reporter) reportWidths(rows []tableRow, hasLastKernel bool) (countW, nameW, valueW, lastKernelW int) {
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

func (r Reporter) buildBorder(countW, nameW, valueW, lastKernelW int, hasLastKernel bool) string {
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

func (r Reporter) buildReportHeader(countW, nameW, valueW, lastKernelW int, hasLastKernel bool, border string) string {
	var h string
	if r.outputFormat == FormatMarkdown || r.outputFormat == FormatGemtext {
		h = strings.Repeat("#", int(r.headerIndent)) + " "
	}
	h += fmt.Sprintf("Top %d %s's by %s\n\n", r.limit, r.metric, r.category)
	desc := MetricDescription(r.metric)
	lineLimit := len(border)
	if r.outputFormat == FormatPlaintext && lineLimit > 0 && len(desc) > lineLimit-1 {
		desc = " " + wordWrap(desc, lineLimit-1)
	}
	h += desc + "\n\n"
	if r.outputFormat == FormatMarkdown || r.outputFormat == FormatGemtext {
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

func (r Reporter) buildFormatStr(countW, nameW, valueW, lastKernelW int, hasLastKernel bool) string {
	if hasLastKernel {
		return fmt.Sprintf("| %%%ds | %%%ds | %%%ds | %%%ds |", countW, nameW, valueW, lastKernelW)
	}
	return fmt.Sprintf("| %%%ds | %%%ds | %%%ds |", countW, nameW, valueW)
}

func (r Reporter) buildReportBody(rows []tableRow, fmtStr string, hasLastKernel bool) string {
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
