package storage

import (
	"errors"
	"sync"
	"time"
)

// Link is what a short code points to.
type Link struct {
	URL string
	// ExpiresAt is when the link stops working. The zero value means it never expires.
	ExpiresAt time.Time
}

// Expired reports whether the link has expired at the given time.
func (l Link) Expired(now time.Time) bool {
	return !l.ExpiresAt.IsZero() && !now.Before(l.ExpiresAt)
}

type Storage interface {
	Save(code string, link Link) error
	Get(code string) (Link, error)
}

var (
	ErrNotFound   = errors.New("not found")
	ErrCodeExists = errors.New("code already exists")
)

type Memory struct {
	mu    sync.RWMutex
	links map[string]Link
}

func NewMemory() *Memory {
	return &Memory{
		links: make(map[string]Link),
	}
}

func (m *Memory) Save(code string, link Link) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.links[code]; ok {
		return ErrCodeExists
	}

	m.links[code] = link
	return nil
}

func (m *Memory) Get(code string) (Link, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	link, ok := m.links[code]
	if !ok {
		return Link{}, ErrNotFound
	}

	return link, nil
}
