package goprecords

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
)

// classLegend explains the short host classification letters below each table
// that shows the Cls column.
const classLegend = "Cls: S = server, W = workstation/laptop, H = hybrid, U = unknown."

// reportColumn is one column of a report table: its header and how to read the
// matching value out of a row.
type reportColumn struct {
	header string
	value  func(tableRow) string
}

// columns returns the columns of the table in display order. Optional columns
// are only included when the table carries data for them.
func (r reportBuilder) columns(t reportTable) []reportColumn {
	cols := []reportColumn{
		{"Pos", func(row tableRow) string { return row.Pos }},
		{r.category.String(), func(row tableRow) string { return row.Name }},
	}
	if t.hasClass {
		cols = append(cols, reportColumn{"Cls", func(row tableRow) string { return row.Class }})
	}
	cols = append(cols, reportColumn{r.metric.String(), func(row tableRow) string { return row.Value }})
	if t.hasLastKernel {
		cols = append(cols, reportColumn{"Last Kernel", func(row tableRow) string { return row.LastKernel }})
	}
	if t.hasLastUpdated {
		cols = append(cols, reportColumn{"Updated", func(row tableRow) string { return row.LastUpdated }})
	}
	return cols
}

// description is the text below the report heading: the metric description plus
// the classification legend when the table shows the Cls column.
func (r reportBuilder) description(t reportTable) string {
	desc := MetricDescription(r.metric)
	if !t.hasClass {
		return desc
	}
	if desc == "" {
		return classLegend
	}
	return desc + " " + classLegend
}

func (r reportBuilder) formatReport(t reportTable, outputFormat OutputFormat) string {
	cols := r.columns(t)
	widths := columnWidths(cols, t.rows)
	border := borderLine(widths)

	var b strings.Builder
	b.WriteString(r.reportHeading(t, outputFormat, len(border)))
	b.WriteString(border)
	b.WriteString(rowLine(headerValues(cols), widths))
	b.WriteString(border)
	for _, row := range t.rows {
		b.WriteString(rowLine(rowValues(cols, row), widths))
	}
	b.WriteString(border)
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		b.WriteString("```\n")
	}
	return b.String()
}

// reportHeading renders the title, the description and (for Markdown/Gemtext)
// the opening code fence. lineLimit is the table width used to wrap the
// plaintext description.
func (r reportBuilder) reportHeading(t reportTable, outputFormat OutputFormat, lineLimit int) string {
	var b strings.Builder
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		b.WriteString(strings.Repeat("#", int(r.headerIndent)))
		b.WriteString(" ")
	}
	fmt.Fprintf(&b, "Top %d %s's by %s\n\n", r.limit, r.metric, r.category)
	desc := r.description(t)
	if outputFormat == FormatPlaintext && lineLimit > 0 && len(desc) > lineLimit-1 {
		desc = " " + wordWrap(desc, lineLimit-1)
	}
	b.WriteString(desc)
	b.WriteString("\n\n")
	if outputFormat == FormatMarkdown || outputFormat == FormatGemtext {
		b.WriteString("```\n")
	}
	return b.String()
}

func (r reportBuilder) formatReportHTML(t reportTable) string {
	hl := int(r.headerIndent)
	if hl < 1 {
		hl = 1
	}
	if hl > 6 {
		hl = 6
	}
	title := fmt.Sprintf("Top %d %s's by %s", r.limit, r.metric, r.category)
	htag := strconv.Itoa(hl)

	var b strings.Builder
	fmt.Fprintf(&b, "<h%s>%s</h%s>\n", htag, template.HTMLEscapeString(title), htag)
	if desc := r.description(t); desc != "" {
		fmt.Fprintf(&b, "<p>%s</p>\n", template.HTMLEscapeString(desc))
	}
	b.WriteString(htmlTable(r.columns(t), t.rows))
	return b.String()
}

func htmlTable(cols []reportColumn, rows []tableRow) string {
	var b strings.Builder
	b.WriteString("<table>\n<tr>")
	for _, c := range cols {
		fmt.Fprintf(&b, "<th>%s</th>", template.HTMLEscapeString(c.header))
	}
	b.WriteString("</tr>\n")
	for _, row := range rows {
		b.WriteString("<tr>")
		for _, v := range rowValues(cols, row) {
			fmt.Fprintf(&b, "<td>%s</td>", template.HTMLEscapeString(v))
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</table>\n")
	return b.String()
}

// columnWidths returns the printed width of each column: the widest of its
// header and all its values.
func columnWidths(cols []reportColumn, rows []tableRow) []int {
	widths := make([]int, len(cols))
	for i, c := range cols {
		widths[i] = len(c.header)
	}
	for _, row := range rows {
		for i, c := range cols {
			if n := len(c.value(row)); n > widths[i] {
				widths[i] = n
			}
		}
	}
	return widths
}

func headerValues(cols []reportColumn) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.header
	}
	return out
}

func rowValues(cols []reportColumn, row tableRow) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = c.value(row)
	}
	return out
}

func borderLine(widths []int) string {
	var b strings.Builder
	for _, w := range widths {
		b.WriteString("+")
		b.WriteString(strings.Repeat("-", 2+w))
	}
	b.WriteString("+\n")
	return b.String()
}

// rowLine renders one right-aligned, padded table row.
func rowLine(values []string, widths []int) string {
	var b strings.Builder
	for i, v := range values {
		fmt.Fprintf(&b, "| %*s ", widths[i], v)
	}
	b.WriteString("|\n")
	return b.String()
}
