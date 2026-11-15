package storage

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

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

// testStorage is the contract every Storage must follow. Memory, Postgres and
// Cached all run it, so they are guaranteed to behave the same way.
// newStore must return an empty storage each time it is called.
func testStorage(t *testing.T, newStore func(t *testing.T) Storage) {
	t.Run("save and get", func(t *testing.T) {
		s := newStore(t)

		if err := s.Save(t.Context(), "abc", Link{URL: "https://go.dev"}); err != nil {
			t.Fatalf("Save: %v", err)
		}

		got, err := s.Get(t.Context(), "abc")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		if got.URL != "https://go.dev" {
			t.Errorf("URL = %q, want %q", got.URL, "https://go.dev")
		}
		if !got.ExpiresAt.IsZero() {
			t.Errorf("ExpiresAt = %v, want zero (never expires)", got.ExpiresAt)
		}
	})

	t.Run("keeps expires_at", func(t *testing.T) {
		s := newStore(t)
		expiresAt := time.Now().UTC().Add(time.Hour).Truncate(time.Second)

		if err := s.Save(t.Context(), "abc", Link{URL: "https://go.dev", ExpiresAt: expiresAt}); err != nil {
			t.Fatal(err)
		}

		got, err := s.Get(t.Context(), "abc")
		if err != nil {
			t.Fatal(err)
		}
		if !got.ExpiresAt.Equal(expiresAt) {
			t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, expiresAt)
		}
	})

	t.Run("get unknown code", func(t *testing.T) {
		s := newStore(t)

		_, err := s.Get(t.Context(), "missing")

		if !errors.Is(err, ErrNotFound) {
			t.Errorf("err = %v, want %v", err, ErrNotFound)
		}
	})

	t.Run("save duplicate code", func(t *testing.T) {
		s := newStore(t)
		if err := s.Save(t.Context(), "abc", Link{URL: "https://first.example"}); err != nil {
			t.Fatal(err)
		}

		err := s.Save(t.Context(), "abc", Link{URL: "https://second.example"})

		if !errors.Is(err, ErrCodeExists) {
			t.Errorf("err = %v, want %v", err, ErrCodeExists)
		}
		// The first URL must not be overwritten.
		if got, _ := s.Get(t.Context(), "abc"); got.URL != "https://first.example" {
			t.Errorf("URL = %q, want %q", got.URL, "https://first.example")
		}
	})

	t.Run("count clicks", func(t *testing.T) {
		s := newStore(t)
		if err := s.Save(t.Context(), "abc", Link{URL: "https://go.dev"}); err != nil {
			t.Fatal(err)
		}

		if got, err := s.Clicks(t.Context(), "abc"); err != nil || got != 0 {
			t.Fatalf("Clicks before any visit = %d, %v; want 0, nil", got, err)
		}
		for range 3 {
			if err := s.IncrementClicks(t.Context(), "abc"); err != nil {
				t.Fatalf("IncrementClicks: %v", err)
			}
		}

		got, err := s.Clicks(t.Context(), "abc")
		if err != nil {
			t.Fatal(err)
		}
		if got != 3 {
			t.Errorf("Clicks = %d, want 3", got)
		}
	})

	t.Run("clicks of unknown code", func(t *testing.T) {
		s := newStore(t)

		if err := s.IncrementClicks(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
			t.Errorf("IncrementClicks err = %v, want %v", err, ErrNotFound)
		}
		if _, err := s.Clicks(t.Context(), "missing"); !errors.Is(err, ErrNotFound) {
			t.Errorf("Clicks err = %v, want %v", err, ErrNotFound)
		}
	})

	// Run with `go test -race`: concurrent use must not lose writes.
	t.Run("concurrent access", func(t *testing.T) {
		s := newStore(t)
		if err := s.Save(t.Context(), "shared", Link{URL: "https://go.dev"}); err != nil {
			t.Fatal(err)
		}
		var wg sync.WaitGroup

		for i := range 50 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				code := fmt.Sprintf("code%d", i)
				if err := s.Save(t.Context(), code, Link{URL: "https://go.dev"}); err != nil {
					t.Errorf("Save(%q): %v", code, err)
				}
				if _, err := s.Get(t.Context(), code); err != nil {
					t.Errorf("Get(%q): %v", code, err)
				}
				if err := s.IncrementClicks(t.Context(), "shared"); err != nil {
					t.Errorf("IncrementClicks: %v", err)
				}
			}()
		}
		wg.Wait()

		if got, _ := s.Clicks(t.Context(), "shared"); got != 50 {
			t.Errorf("Clicks = %d, want 50", got)
		}
	})
}
