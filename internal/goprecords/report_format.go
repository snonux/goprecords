package goprecords

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

func (r reportBuilder) formatReport(rows []tableRow, hasLastKernel, hasLastUpdated bool, outputFormat OutputFormat) string {
	cW, nW, vW, lkW, luW := r.reportWidths(rows, hasLastKernel, hasLastUpdated)
	border := r.buildBorder(cW, nW, vW, lkW, luW, hasLastKernel, hasLastUpdated)
	header := r.buildReportHeader(cW, nW, vW, lkW, luW, hasLastKernel, hasLastUpdated, border, outputFormat)
	fmtStr := r.buildFormatStr(cW, nW, vW, lkW, luW, hasLastKernel, hasLastUpdated)
	body := r.buildReportBody(rows, fmtStr, hasLastKernel, hasLastUpdated)
	out := header + body + border
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		out += "```\n"
	}
	return out
}

func (r reportBuilder) formatReportHTML(rows []tableRow, hasLastKernel, hasLastUpdated bool) string {
	hl := int(r.headerIndent)
	if hl < 1 {
		hl = 1
	}
	if hl > 6 {
		hl = 6
	}
	title := fmt.Sprintf("Top %d %s's by %s", r.limit, r.metric, r.category)
	desc := MetricDescription(r.metric)
	htag := strconv.Itoa(hl)

	var b strings.Builder
	b.WriteString("<h")
	b.WriteString(htag)
	b.WriteString(">")
	b.WriteString(template.HTMLEscapeString(title))
	b.WriteString("</h")
	b.WriteString(htag)
	b.WriteString(">\n")
	if desc != "" {
		b.WriteString("<p>")
		b.WriteString(template.HTMLEscapeString(desc))
		b.WriteString("</p>\n")
	}
	b.WriteString("<table>\n<tr><th>Pos</th><th>")
	b.WriteString(template.HTMLEscapeString(r.category.String()))
	b.WriteString("</th><th>")
	b.WriteString(template.HTMLEscapeString(r.metric.String()))
	b.WriteString("</th>")
	if hasLastKernel {
		b.WriteString("<th>Last Kernel</th>")
	}
	if hasLastUpdated {
		b.WriteString("<th>Updated</th>")
	}
	b.WriteString("</tr>\n")
	for _, row := range rows {
		b.WriteString("<tr><td>")
		b.WriteString(template.HTMLEscapeString(row.Pos))
		b.WriteString("</td><td>")
		b.WriteString(template.HTMLEscapeString(row.Name))
		b.WriteString("</td><td>")
		b.WriteString(template.HTMLEscapeString(row.Value))
		b.WriteString("</td>")
		if hasLastKernel {
			b.WriteString("<td>")
			b.WriteString(template.HTMLEscapeString(row.LastKernel))
			b.WriteString("</td>")
		}
		if hasLastUpdated {
			b.WriteString("<td>")
			b.WriteString(template.HTMLEscapeString(row.LastUpdated))
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String()
}

func (r reportBuilder) reportWidths(rows []tableRow, hasLastKernel, hasLastUpdated bool) (countW, nameW, valueW, lastKernelW, lastUpdatedW int) {
	countW = 3
	nameW = len(r.category.String())
	valueW = len(r.metric.String())
	if hasLastKernel {
		lastKernelW = len("Last Kernel")
	}
	if hasLastUpdated {
		lastUpdatedW = len("Updated")
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
		if len(row.LastUpdated) > lastUpdatedW {
			lastUpdatedW = len(row.LastUpdated)
		}
	}
	return countW, nameW, valueW, lastKernelW, lastUpdatedW
}

func (r reportBuilder) buildBorder(countW, nameW, valueW, lastKernelW, lastUpdatedW int, hasLastKernel, hasLastUpdated bool) string {
	parts := []string{
		"+" + strings.Repeat("-", 2+countW),
		"+" + strings.Repeat("-", 2+nameW),
		"+" + strings.Repeat("-", 2+valueW),
	}
	if hasLastKernel {
		parts = append(parts, "+"+strings.Repeat("-", 2+lastKernelW))
	}
	if hasLastUpdated {
		parts = append(parts, "+"+strings.Repeat("-", 2+lastUpdatedW))
	}
	return strings.Join(parts, "") + "+\n"
}

func (r reportBuilder) buildReportHeader(countW, nameW, valueW, lastKernelW, lastUpdatedW int, hasLastKernel, hasLastUpdated bool, border string, outputFormat OutputFormat) string {
	var b strings.Builder
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		b.WriteString(strings.Repeat("#", int(r.headerIndent)))
		b.WriteString(" ")
	}
	b.WriteString(fmt.Sprintf("Top %d %s's by %s\n\n", r.limit, r.metric, r.category))
	desc := MetricDescription(r.metric)
	lineLimit := len(border)
	if outputFormat == FormatPlaintext && lineLimit > 0 && len(desc) > lineLimit-1 {
		desc = " " + wordWrap(desc, lineLimit-1)
	}
	b.WriteString(desc)
	b.WriteString("\n\n")
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		b.WriteString("```\n")
	}
	b.WriteString(border)
	fmtStr := r.buildFormatStr(countW, nameW, valueW, lastKernelW, lastUpdatedW, hasLastKernel, hasLastUpdated)
	if hasLastKernel && hasLastUpdated {
		b.WriteString(fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String(), "Last Kernel", "Updated"))
	} else if hasLastKernel {
		b.WriteString(fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String(), "Last Kernel"))
	} else if hasLastUpdated {
		b.WriteString(fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String(), "Updated"))
	} else {
		b.WriteString(fmt.Sprintf(fmtStr+"\n", "Pos", r.category.String(), r.metric.String()))
	}
	b.WriteString(border)
	return b.String()
}

func (r reportBuilder) buildFormatStr(countW, nameW, valueW, lastKernelW, lastUpdatedW int, hasLastKernel, hasLastUpdated bool) string {
	if hasLastKernel && hasLastUpdated {
		return fmt.Sprintf("| %%%ds | %%%ds | %%%ds | %%%ds | %%%ds |", countW, nameW, valueW, lastKernelW, lastUpdatedW)
	}
	if hasLastKernel {
		return fmt.Sprintf("| %%%ds | %%%ds | %%%ds | %%%ds |", countW, nameW, valueW, lastKernelW)
	}
	if hasLastUpdated {
		return fmt.Sprintf("| %%%ds | %%%ds | %%%ds | %%%ds |", countW, nameW, valueW, lastUpdatedW)
	}
	return fmt.Sprintf("| %%%ds | %%%ds | %%%ds |", countW, nameW, valueW)
}

func (r reportBuilder) buildReportBody(rows []tableRow, fmtStr string, hasLastKernel, hasLastUpdated bool) string {
	var b strings.Builder
	for _, row := range rows {
		if hasLastKernel && hasLastUpdated {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value, row.LastKernel, row.LastUpdated))
		} else if hasLastKernel {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value, row.LastKernel))
		} else if hasLastUpdated {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value, row.LastUpdated))
		} else {
			b.WriteString(fmt.Sprintf(fmtStr+"\n", row.Pos, row.Name, row.Value))
		}
	}
	return b.String()
}
