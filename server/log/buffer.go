package log

import (
	"log/slog"
	"sync"
	"time"
)

type LogEntry struct {
	Time  time.Time
	Level slog.Level
	Msg   string
	Attrs map[string]any
}

type RingBuffer struct {
	mu      sync.RWMutex
	entries []LogEntry
	size    int
	head    int
	count   int
}

func NewRingBuffer(size int) *RingBuffer {
	return &RingBuffer{
		entries: make([]LogEntry, size),
		size:    size,
	}
}

func (rb *RingBuffer) Add(entry LogEntry) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.entries[rb.head] = entry
	rb.head = (rb.head + 1) % rb.size
	if rb.count < rb.size {
		rb.count++
	}
}

func (rb *RingBuffer) GetSince(t time.Time) []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]LogEntry, 0)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.size) % rb.size
		entry := rb.entries[idx]
		if entry.Time.After(t) {
			result = append(result, entry)
		}
	}
	return result
}

func (rb *RingBuffer) GetAll() []LogEntry {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	result := make([]LogEntry, rb.count)
	for i := 0; i < rb.count; i++ {
		idx := (rb.head - rb.count + i + rb.size) % rb.size
		result[i] = rb.entries[idx]
	}
	return result
}
