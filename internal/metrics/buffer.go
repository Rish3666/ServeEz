package metrics

import (
	"sync"
)

type Buffer struct {
	mu   sync.Mutex
	buf  []Sample
	head int
	size int
}

func NewBuffer(minutes int, intervalNanos int64) *Buffer {
	capacity := minutes * 60
	if intervalNanos > 0 {
		capacity = int((int64(minutes) * 60 * 1_000_000_000) / intervalNanos)
	}
	if capacity < 1 {
		capacity = 1
	}
	return &Buffer{buf: make([]Sample, capacity)}
}

func (b *Buffer) Add(v Sample) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if len(b.buf) == 0 {
		return
	}
	b.buf[b.head] = v
	b.head = (b.head + 1) % len(b.buf)
	if b.size < len(b.buf) {
		b.size++
	}
}

func (b *Buffer) Snapshot() []Sample {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.snapshotLocked()
}

func (b *Buffer) Drain() []Sample {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := b.snapshotLocked()
	b.head = 0
	b.size = 0
	return out
}

func (b *Buffer) snapshotLocked() []Sample {
	out := make([]Sample, 0, b.size)
	if b.size == 0 {
		return out
	}
	start := b.head - b.size
	if start < 0 {
		start += len(b.buf)
	}
	for i := 0; i < b.size; i++ {
		out = append(out, b.buf[(start+i)%len(b.buf)])
	}
	return out
}
