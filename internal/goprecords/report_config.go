package goprecords

import (
	"flag"
	"fmt"
	"net/url"
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
		metric:        fs.String("metric", "Uptime", "Metric: Boots, Uptime, Score, Downtime, Lifespan, LastUpdated"),
		limit:         fs.Uint("limit", 20, "Limit output to num of entries"),
		outputFormat:  fs.String("output-format", "Plaintext", "Output format: Plaintext, Markdown, Gemtext, HTML"),
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
// output-format, all, include-kernel, stats-order). It also accepts Category,
// Metric, and OutputFormat as alternate keys (same values as the CLI).
func ParseReportQuery(q url.Values) (ReportConfig, error) {
	cat, err := parseReportQueryCategory(q)
	if err != nil {
		return ReportConfig{}, err
	}
	met, err := parseReportQueryMetric(q)
	if err != nil {
		return ReportConfig{}, err
	}
	limit, err := parseReportQueryLimit(q)
	if err != nil {
		return ReportConfig{}, err
	}
	outFmt, err := parseReportQueryOutputFormat(q)
	if err != nil {
		return ReportConfig{}, err
	}
	all, err := parseReportQueryOptionalBool(q, "all")
	if err != nil {
		return ReportConfig{}, err
	}
	includeKernel, err := parseReportQueryOptionalBool(q, "include-kernel")
	if err != nil {
		return ReportConfig{}, err
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

func parseReportQueryCategory(q url.Values) (Category, error) {
	s := firstQuery(q, "category", "Category")
	if s == "" {
		s = "Host"
	}
	return ParseCategory(s)
}

func parseReportQueryMetric(q url.Values) (Metric, error) {
	s := firstQuery(q, "metric", "Metric")
	if s == "" {
		s = "Uptime"
	}
	return ParseMetric(s)
}

func parseReportQueryLimit(q url.Values) (uint, error) {
	if ls := q.Get("limit"); ls != "" {
		v, err := strconv.ParseUint(ls, 10, 32)
		if err != nil {
			return 0, fmt.Errorf("invalid limit %q", ls)
		}
		return uint(v), nil
	}
	return 20, nil
}

func parseReportQueryOutputFormat(q url.Values) (OutputFormat, error) {
	s := firstQuery(q, "output-format", "OutputFormat")
	if s == "" {
		s = "Plaintext"
	}
	return ParseOutputFormat(s)
}

func parseReportQueryOptionalBool(q url.Values, key string) (bool, error) {
	v := q.Get(key)
	if v == "" {
		return false, nil
	}
	return parseQueryBool(v)
}

func firstQuery(q url.Values, keys ...string) string {
	for _, k := range keys {
		if v := q.Get(k); v != "" {
			return v
		}
	}
	return ""
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
