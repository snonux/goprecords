package goprecords

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

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

func (r reportBuilder) formatReportHTML(rows []tableRow, hasLastKernel bool) string {
	cW, nW, vW, lkW := r.reportWidths(rows, hasLastKernel)
	border := r.buildBorder(cW, nW, vW, lkW, hasLastKernel)
	fmtStr := r.buildFormatStr(cW, nW, vW, lkW, hasLastKernel)
	var headRow string
	if hasLastKernel {
		headRow = fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String(), "Last Kernel")
	} else {
		headRow = fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String())
	}
	body := r.buildReportBody(rows, fmtStr, hasLastKernel)
	ascii := border + headRow + border + body + border

	hl := int(r.headerIndent)
	if hl < 1 {
		hl = 1
	}
	if hl > 6 {
		hl = 6
	}
	title := fmt.Sprintf("Top %d %s's by %s", r.limit, r.metric, r.category)
	desc := MetricDescription(r.metric)

	var b strings.Builder
	b.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n<meta charset=\"utf-8\">\n<title>")
	b.WriteString(template.HTMLEscapeString(title))
	b.WriteString("</title>\n</head>\n<body>\n<h")
	b.WriteString(strconv.Itoa(hl))
	b.WriteString(">")
	b.WriteString(template.HTMLEscapeString(title))
	b.WriteString("</h")
	b.WriteString(strconv.Itoa(hl))
	b.WriteString(">\n")
	if desc != "" {
		b.WriteString("<p>")
		b.WriteString(template.HTMLEscapeString(desc))
		b.WriteString("</p>\n")
	}
	b.WriteString("<pre>")
	b.WriteString(template.HTMLEscapeString(ascii))
	b.WriteString("</pre>\n</body>\n</html>\n")
	return b.String()
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
