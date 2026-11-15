package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"
)

func TestMemorySaveAndGet(t *testing.T) {
	m := NewMemory()

	if err := m.Save("abc", "https://go.dev"); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := m.Get("abc")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "https://go.dev" {
		t.Errorf("Get = %q, want %q", got, "https://go.dev")
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
	if err := m.Save("abc", "https://first.example"); err != nil {
		t.Fatal(err)
	}

	err := m.Save("abc", "https://second.example")

	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("err = %v, want %v", err, ErrCodeExists)
	}
	// The first URL must not be overwritten.
	if got, _ := m.Get("abc"); got != "https://first.example" {
		t.Errorf("Get = %q, want %q", got, "https://first.example")
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
			if err := m.Save(code, "https://go.dev"); err != nil {
				t.Errorf("Save(%q): %v", code, err)
			}
			if _, err := m.Get(code); err != nil {
				t.Errorf("Get(%q): %v", code, err)
			}
		}()
	}

	wg.Wait()
}
