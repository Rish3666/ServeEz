package metrics

import (
	"context"
	"errors"
	"math"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Rish3666/ServeEz/internal/api"
	"golang.org/x/sys/unix"
)

type Sample struct {
	At       time.Time
	Usage    api.Usage
	Hardware *api.HardwareInfo
}

type Collector struct {
	root string
}

var (
	readCPUPercent  = cpuPercent
	readMemPercent  = memPercent
	readDiskPercent = diskPercent
	readNetBps      = netBps
)

func NewCollector(root string) *Collector {
	if root == "" {
		root = "/"
	}
	return &Collector{root: root}
}

func (c *Collector) Collect(ctx context.Context) (Sample, error) {
	if err := ctx.Err(); err != nil {
		return Sample{}, err
	}
	cpu, err := readCPUPercent()
	if err != nil {
		return Sample{}, err
	}
	mem, err := readMemPercent()
	if err != nil {
		return Sample{}, err
	}
	disk, err := readDiskPercent(c.root)
	if err != nil {
		return Sample{}, err
	}
	rx, tx, err := readNetBps()
	if err != nil {
		return Sample{}, err
	}
	return Sample{
		At: time.Now().UTC(),
		Usage: api.Usage{
			CPUPercent:  cpu,
			MemPercent:  mem,
			DiskPercent: disk,
			NetRxBps:    rx,
			NetTxBps:    tx,
		},
	}, nil
}

func cpuPercent() (float64, error) {
	text, err := unix.Sysctl("vm.loadavg")
	if err != nil {
		return 0, err
	}
	fields := floatFields(text)
	if len(fields) == 0 {
		return 0, errors.New("cpu load unavailable")
	}
	load := fields[0]
	cpu := float64(runtime.NumCPU())
	if cpu < 1 {
		cpu = 1
	}
	return clamp((load / cpu) * 100), nil
}

func memPercent() (float64, error) {
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil {
		return 0, err
	}
	pageSize, err := unix.SysctlUint64("hw.pagesize")
	if err != nil {
		return 0, err
	}
	free, _ := unix.SysctlUint64("vm.page_free_count")
	inactive, _ := unix.SysctlUint64("vm.page_inactive_count")
	speculative, _ := unix.SysctlUint64("vm.page_speculative_count")
	freeBytes := (free + inactive + speculative) * pageSize
	if freeBytes > total {
		freeBytes = total
	}
	used := total - freeBytes
	if total == 0 {
		return 0, errors.New("memory size unavailable")
	}
	return clamp((float64(used) / float64(total)) * 100), nil
}

func diskPercent(path string) (float64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return 0, err
	}
	if stat.Blocks == 0 {
		return 0, errors.New("disk stats unavailable")
	}
	used := float64(stat.Blocks-stat.Bfree) / float64(stat.Blocks) * 100
	return clamp(used), nil
}

func netBps() (uint64, uint64, error) {
	return 0, 0, nil
}

var floatRe = regexp.MustCompile(`[-+]?[0-9]*\.?[0-9]+`)

func floatFields(s string) []float64 {
	matches := floatRe.FindAllString(s, -1)
	out := make([]float64, 0, len(matches))
	for _, m := range matches {
		v, err := strconv.ParseFloat(strings.TrimSpace(m), 64)
		if err == nil {
			out = append(out, v)
		}
	}
	return out
}

func clamp(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
