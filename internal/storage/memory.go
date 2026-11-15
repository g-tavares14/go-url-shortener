package storage

import (
	"context"
	"sync"
)

// Memory keeps everything in a map. Data is lost when the process stops, so
// it is meant for development and tests.
type Memory struct {
	mu      sync.RWMutex
	entries map[string]memoryEntry
}

type memoryEntry struct {
	link   Link
	clicks int64
}

func NewMemory() *Memory {
	return &Memory{
		entries: make(map[string]memoryEntry),
	}
}

func (m *Memory) Save(ctx context.Context, code string, link Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.entries[code]; ok {
		return ErrCodeExists
	}

	m.entries[code] = memoryEntry{link: link}
	return nil
}

func (m *Memory) Get(ctx context.Context, code string) (Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[code]
	if !ok {
		return Link{}, ErrNotFound
	}

	return entry.link, nil
}

func (m *Memory) IncrementClicks(ctx context.Context, code string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.entries[code]
	if !ok {
		return ErrNotFound
	}

	entry.clicks++
	m.entries[code] = entry
	return nil
}

func (m *Memory) Clicks(ctx context.Context, code string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.entries[code]
	if !ok {
		return 0, ErrNotFound
	}

	return entry.clicks, nil
}
