package recordline

import (
	"strconv"
	"strings"
)

// Fields holds the values parsed from a single uptimed record line.
type Fields struct {
	Uptime      uint64
	BootTime    uint64
	OS          string
	KernelName  string
	KernelMajor string
}

// Parse parses one non-empty line of the form "uptime:boottime:os..." from an
// uptimed .records file. It returns false if the line is empty or malformed.
func Parse(line string) (Fields, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return Fields{}, false
	}
	parts := strings.SplitN(line, ":", 3)
	if len(parts) != 3 {
		return Fields{}, false
	}
	uptime, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return Fields{}, false
	}
	bootTime, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return Fields{}, false
	}
	osStr := parts[2]
	kernelName := osStr
	if i := strings.Index(osStr, " "); i > 0 {
		kernelName = osStr[:i]
	}
	kernelMajor := kernelName + " "
	rest := osStr
	if i := strings.Index(osStr, " "); i >= 0 {
		rest = osStr[i+1:]
	}
	if j := strings.Index(rest, "."); j >= 0 {
		kernelMajor += rest[:j] + "..."
	} else {
		kernelMajor += rest + "..."
	}
	return Fields{
		Uptime:      uptime,
		BootTime:    bootTime,
		OS:          osStr,
		KernelName:  kernelName,
		KernelMajor: kernelMajor,
	}, true
}
