package goprecords

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
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
		lastKernel, err := lastKernelFromFile(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("last kernel %s: %w", path, err)
		}
		out.Host[host] = NewHostAggregate(host, lastKernel)
		if err := processRecordsFile(ctx, path, host, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func processRecordsFile(ctx context.Context, path, host string, out *Aggregates) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec, ok := parseRecordLine(sc.Text())
		if !ok {
			continue
		}
		out.Host[host].AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.Kernel, rec.OS).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelName, rec.KernelName).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelMajor, rec.KernelMajor).AddRecord(rec.Uptime, rec.BootTime)
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", path, err)
	}
	return nil
}

func lastKernelFromFile(ctx context.Context, path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	var maxBoot uint64
	var lastOS string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		default:
		}
		rec, ok := parseRecordLine(sc.Text())
		if !ok {
			continue
		}
		if rec.BootTime >= maxBoot {
			maxBoot = rec.BootTime
			lastOS = rec.OS
		}
	}
	return lastOS, sc.Err()
}
