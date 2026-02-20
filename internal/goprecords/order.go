package goprecords

import (
	"fmt"
	"strings"
)

// CategoryMetric pairs a category with a metric for stats order.
type CategoryMetric struct {
	Category Category
	Metric   Metric
}

// ParseStatsOrder parses a comma-separated "Category:Metric" list.
func ParseStatsOrder(s string) ([]CategoryMetric, error) {
	parts := strings.Split(s, ",")
	var entries []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			entries = append(entries, p)
		}
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("invalid -stats-order: empty list")
	}
	var order []CategoryMetric
	seen := make(map[string]bool)
	for _, entry := range entries {
		idx := strings.Index(entry, ":")
		if idx <= 0 || idx == len(entry)-1 {
			return nil, fmt.Errorf("invalid -stats-order entry %q (expected Category:Metric)", entry)
		}
		catName := strings.TrimSpace(entry[:idx])
		metName := strings.TrimSpace(entry[idx+1:])
		if catName == "" || metName == "" {
			return nil, fmt.Errorf("invalid -stats-order entry %q (expected Category:Metric)", entry)
		}
		cat, err := ParseCategory(catName)
		if err != nil {
			return nil, fmt.Errorf("invalid -stats-order category %q", catName)
		}
		met, err := ParseMetric(metName)
		if err != nil {
			return nil, fmt.Errorf("invalid -stats-order metric %q", metName)
		}
		if cat != CategoryHost && (met == MetricDowntime || met == MetricLifespan) {
			return nil, fmt.Errorf("invalid -stats-order entry %q (metric %s not supported for category %s)", entry, metName, catName)
		}
		key := cat.String() + ":" + met.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		order = append(order, CategoryMetric{cat, met})
	}
	return order, nil
}

// StatsOrderList returns the full order (custom entries first, then default remainder).
func StatsOrderList(statsOrder string) ([]CategoryMetric, error) {
	defaultOrder := defaultStatsOrder()
	if statsOrder == "" {
		return defaultOrder, nil
	}
	order, err := ParseStatsOrder(statsOrder)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool)
	for _, p := range order {
		seen[p.Category.String()+":"+p.Metric.String()] = true
	}
	for _, p := range defaultOrder {
		key := p.Category.String() + ":" + p.Metric.String()
		if seen[key] {
			continue
		}
		seen[key] = true
		order = append(order, p)
	}
	return order, nil
}

func defaultStatsOrder() []CategoryMetric {
	var out []CategoryMetric
	for _, c := range []Category{CategoryHost, CategoryKernel, CategoryKernelMajor, CategoryKernelName} {
		for _, m := range []Metric{MetricBoots, MetricUptime, MetricScore, MetricDowntime, MetricLifespan} {
			out = append(out, CategoryMetric{c, m})
		}
	}
	return out
}
