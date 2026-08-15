package metrics

import (
	"context"
	"testing"
)

func TestCollectorCollect(t *testing.T) {
	oldCPU, oldMem, oldDisk, oldNet := readCPUPercent, readMemPercent, readDiskPercent, readNetBps
	t.Cleanup(func() {
		readCPUPercent = oldCPU
		readMemPercent = oldMem
		readDiskPercent = oldDisk
		readNetBps = oldNet
	})
	readCPUPercent = func() (float64, error) { return 12.5, nil }
	readMemPercent = func() (float64, error) { return 42.0, nil }
	readDiskPercent = func(string) (float64, error) { return 66.0, nil }
	readNetBps = func() (uint64, uint64, error) { return 100, 200, nil }

	sample, err := NewCollector("/").Collect(context.Background())
	if err != nil {
		t.Fatalf("Collect() error = %v", err)
	}
	if sample.Usage.CPUPercent != 12.5 {
		t.Fatalf("CPUPercent = %v", sample.Usage.CPUPercent)
	}
	if sample.Usage.MemPercent != 42.0 {
		t.Fatalf("MemPercent = %v", sample.Usage.MemPercent)
	}
	if sample.Usage.DiskPercent != 66.0 {
		t.Fatalf("DiskPercent = %v", sample.Usage.DiskPercent)
	}
	if sample.Usage.NetRxBps != 100 || sample.Usage.NetTxBps != 200 {
		t.Fatalf("Net bps = %+v", sample.Usage)
	}
}
