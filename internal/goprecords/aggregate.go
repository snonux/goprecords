package goprecords

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Aggregates holds all category maps. Host uses HostAggregate; others use Aggregate.
type Aggregates struct {
	Host        map[string]*HostAggregate
	Kernel      map[string]*Aggregate
	KernelMajor map[string]*Aggregate
	KernelName  map[string]*Aggregate
}

// Aggregator reads .records files from a directory and builds Aggregates.
type Aggregator struct {
	statsDir string
}

// NewAggregator returns an Aggregator for the given stats directory.
func NewAggregator(statsDir string) *Aggregator {
	return &Aggregator{statsDir: statsDir}
}

// Aggregate reads all .records files and returns aggregated data.
func (ag *Aggregator) Aggregate(ctx context.Context) (*Aggregates, error) {
	out := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	entries, err := os.ReadDir(ag.statsDir)
	if err != nil {
		return nil, fmt.Errorf("read stats dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".records") {
			continue
		}
		path := filepath.Join(ag.statsDir, e.Name())
		info, err := os.Stat(path)
		if err != nil || info.Size() == 0 {
			continue
		}
		host := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		if idx := strings.Index(host, "."); idx > 0 {
			host = host[:idx]
		}
		if _, exists := out.Host[host]; exists {
			return nil, fmt.Errorf("record file for %s already processed - duplicate inputs?", host)
		}
		lastKernel, err := lastKernelFromFile(path)
		if err != nil {
			return nil, fmt.Errorf("last kernel %s: %w", path, err)
		}
		out.Host[host] = NewHostAggregate(host, lastKernel)
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open %s: %w", path, err)
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ":", 3)
			if len(parts) != 3 {
				continue
			}
			uptime, _ := strconv.ParseUint(parts[0], 10, 64)
			bootTime, _ := strconv.ParseUint(parts[1], 10, 64)
			osStr := parts[2]
			uname := osStr
			if i := strings.Index(osStr, " "); i > 0 {
				uname = osStr[:i]
			}
			osMajor := uname + " "
			rest := osStr
			if i := strings.Index(osStr, " "); i >= 0 {
				rest = osStr[i+1:]
			}
			if j := strings.Index(rest, "."); j >= 0 {
				osMajor += rest[:j] + "..."
			} else {
				osMajor += rest + "..."
			}
			out.Host[host].AddRecord(uptime, bootTime)
			getOrNewAggregate(out.Kernel, osStr).AddRecord(uptime, bootTime)
			getOrNewAggregate(out.KernelName, uname).AddRecord(uptime, bootTime)
			getOrNewAggregate(out.KernelMajor, osMajor).AddRecord(uptime, bootTime)
		}
		if err := sc.Err(); err != nil {
			return nil, fmt.Errorf("scan %s: %w", path, err)
		}
	}
	return out, nil
}

func getOrNewAggregate(m map[string]*Aggregate, name string) *Aggregate {
	if a, ok := m[name]; ok {
		return a
	}
	a := NewAggregate(name)
	m[name] = a
	return a
}

func lastKernelFromFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var maxBoot uint64
	var lastOS string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		bootTime, _ := strconv.ParseUint(parts[1], 10, 64)
		if bootTime >= maxBoot {
			maxBoot = bootTime
			lastOS = parts[2]
		}
	}
	return lastOS, sc.Err()
}
