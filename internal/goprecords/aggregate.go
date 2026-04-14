package goprecords

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"

	"codeberg.org/snonux/goprecords/internal/recordline"
	"codeberg.org/snonux/goprecords/internal/recordsdir"
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
	fsys fs.FS
	root string
}

// NewAggregator returns an Aggregator for the given stats directory.
func NewAggregator(statsDir string) *Aggregator {
	return NewAggregatorFS(os.DirFS(statsDir))
}

// NewAggregatorFS returns an Aggregator that reads non-empty .records files from the root of fsys.
func NewAggregatorFS(fsys fs.FS) *Aggregator {
	return &Aggregator{fsys: fsys, root: "."}
}

// Aggregate reads all .records files and returns aggregated data.
func (ag *Aggregator) Aggregate(ctx context.Context) (*Aggregates, error) {
	out := &Aggregates{
		Host:        make(map[string]*HostAggregate),
		Kernel:      make(map[string]*Aggregate),
		KernelMajor: make(map[string]*Aggregate),
		KernelName:  make(map[string]*Aggregate),
	}
	files, err := recordsdir.ListNonEmptyFilesFS(ag.fsys, ag.root)
	if err != nil {
		return nil, fmt.Errorf("read stats dir: %w", err)
	}
	for _, f := range files {
		host := f.Host
		relPath := f.Path
		if _, exists := out.Host[host]; exists {
			return nil, fmt.Errorf("record file for %s already processed - duplicate inputs?", host)
		}
		lastKernel, err := lastKernelFromFile(ctx, ag.fsys, relPath)
		if err != nil {
			return nil, fmt.Errorf("last kernel %s: %w", relPath, err)
		}
		out.Host[host] = NewHostAggregate(host, lastKernel)
		if err := processRecordsFile(ctx, ag.fsys, relPath, host, out); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func processRecordsFile(ctx context.Context, fsys fs.FS, relPath, host string, out *Aggregates) error {
	f, err := fsys.Open(relPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", relPath, err)
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		rec, ok := recordline.Parse(sc.Text())
		if !ok {
			continue
		}
		out.Host[host].AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.Kernel, rec.OS).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelName, rec.KernelName).AddRecord(rec.Uptime, rec.BootTime)
		getOrNewAggregate(out.KernelMajor, rec.KernelMajor).AddRecord(rec.Uptime, rec.BootTime)
	}
	if err := sc.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", relPath, err)
	}
	return nil
}

func lastKernelFromFile(ctx context.Context, fsys fs.FS, relPath string) (string, error) {
	f, err := fsys.Open(relPath)
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
		rec, ok := recordline.Parse(sc.Text())
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
