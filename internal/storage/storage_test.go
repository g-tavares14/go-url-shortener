package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestMemorySaveAndGet(t *testing.T) {
	m := NewMemory()

	if err := m.Save("abc", Link{URL: "https://go.dev"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := m.Get("abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.URL != "https://go.dev" {
		t.Errorf("Get().URL = %q, want %q", got.URL, "https://go.dev")
	}
}

func TestMemoryGetUnknownCode(t *testing.T) {
	m := NewMemory()

	_, err := m.Get("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want %v", err, ErrNotFound)
	}
}

func TestMemorySaveDuplicateCode(t *testing.T) {
	m := NewMemory()
	if err := m.Save("abc", Link{URL: "https://first.example"}); err != nil {
		t.Fatal(err)
	}

	err := m.Save("abc", Link{URL: "https://second.example"})

	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("err = %v, want %v", err, ErrCodeExists)
	}
	// The first URL must not be overwritten.
	if got, _ := m.Get("abc"); got.URL != "https://first.example" {
		t.Errorf("Get().URL = %q, want %q", got.URL, "https://first.example")
	}
}

// Run with `go test -race` to make the race detector check the mutex.
func TestMemoryConcurrentAccess(t *testing.T) {
	m := NewMemory()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			code := fmt.Sprintf("code%d", i)
			if err := m.Save(code, Link{URL: "https://go.dev"}); err != nil {
				t.Errorf("Save(%q): %v", code, err)
			}
			if _, err := m.Get(code); err != nil {
				t.Errorf("Get(%q): %v", code, err)
			}
		}()
	}

	wg.Wait()
}

func TestMemorySaveKeepsExpiresAt(t *testing.T) {
	m := NewMemory()
	expiresAt := time.Date(2030, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := m.Save("abc", Link{URL: "https://go.dev", ExpiresAt: expiresAt}); err != nil {
		t.Fatal(err)
	}

	got, err := m.Get("abc")
	if err != nil {
		t.Fatal(err)
	}
	if !got.ExpiresAt.Equal(expiresAt) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
	}
}

func TestLinkExpired(t *testing.T) {
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"never expires", time.Time{}, false},
		{"expires in the future", now.Add(time.Minute), false},
		{"expires exactly now", now, true},
		{"expired in the past", now.Add(-time.Minute), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			link := Link{URL: "https://go.dev", ExpiresAt: tt.expiresAt}
			if got := link.Expired(now); got != tt.want {
				t.Errorf("Expired() = %v, want %v", got, tt.want)
			}
		})
	}
}
