package storage

import (
	"errors"
	"sync"
)

type Storage interface {
	Save(code, url string) error
	Get(code string) (string, error)
}

var (
	ErrNotFound   = errors.New("not found")
	ErrCodeExists = errors.New("code already exists")
)

type Memory struct {
	mu   sync.RWMutex
	urls map[string]string
}

func NewMemory() *Memory {
	return &Memory{
		urls: make(map[string]string),
	}
}

func (m *Memory) Save(code, url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.urls[code]; ok {
		return ErrCodeExists
	}

	m.urls[code] = url
	return nil
}

func (m *Memory) Get(code string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	url, ok := m.urls[code]
	if !ok {
		return "", ErrNotFound
	}

	return url, nil
}
