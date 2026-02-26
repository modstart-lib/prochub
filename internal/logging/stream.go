package logging

import "sync"

type StreamHub struct {
	mu      sync.RWMutex
	entries []Entry
	limit   int
}

func NewStreamHub(limit int) *StreamHub {
	return &StreamHub{
		entries: make([]Entry, 0, limit),
		limit:   limit,
	}
}

func (h *StreamHub) Push(entry Entry) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.entries = append(h.entries, entry)
	if len(h.entries) > h.limit {
		h.entries = h.entries[len(h.entries)-h.limit:]
	}
}

func (h *StreamHub) Snapshot() []Entry {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data := make([]Entry, len(h.entries))
	copy(data, h.entries)
	return data
}
