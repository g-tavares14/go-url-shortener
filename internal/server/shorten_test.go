package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"go-url-shortener/internal/storage"
)

func TestShortenValidURL(t *testing.T) {
	store := storage.NewMemory()
	mux := newTestMux(store)

	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var resp shortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Code) != codeLength {
		t.Errorf("len(code) = %d, want %d", len(resp.Code), codeLength)
	}
	if want := "http://test.local/" + resp.Code; resp.ShortURL != want {
		t.Errorf("short_url = %q, want %q", resp.ShortURL, want)
	}

	got, err := store.Get(t.Context(), resp.Code)
	if err != nil {
		t.Fatalf("code %q was not saved: %v", resp.Code, err)
	}
	if got.URL != "https://go.dev" {
		t.Errorf("saved url = %q, want %q", got.URL, "https://go.dev")
	}
	if resp.ExpiresAt != nil {
		t.Errorf("expires_at = %v, want it omitted", resp.ExpiresAt)
	}
	if !got.ExpiresAt.IsZero() {
		t.Errorf("saved ExpiresAt = %v, want zero (never expires)", got.ExpiresAt)
	}
}

func TestShortenWithExpiresIn(t *testing.T) {
	store := storage.NewMemory()
	mux := newTestMux(store)

	before := time.Now()
	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev","expires_in":3600}`)
	after := time.Now()

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	var resp shortenResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ExpiresAt == nil {
		t.Fatal("expires_at is missing from the response")
	}

	// The handler reads the clock between before and after, so the expiry
	// must fall one hour after that window. the handler truncates to whole seconds,
	// hence the one-second margin.
	earliest := before.Add(time.Hour).Add(-time.Second)
	latest := after.Add(time.Hour).Add(time.Second)
	if resp.ExpiresAt.Before(earliest) || resp.ExpiresAt.After(latest) {
		t.Errorf("expires_at = %v, want between %v and %v", resp.ExpiresAt, earliest, latest)
	}

	got, err := store.Get(t.Context(), resp.Code)
	if err != nil {
		t.Fatalf("code %q was not saved: %v", resp.Code, err)
	}
	if !got.ExpiresAt.Equal(*resp.ExpiresAt) {
		t.Errorf("saved ExpiresAt = %v, response says %v", got.ExpiresAt, *resp.ExpiresAt)
	}
}

func TestShortenBadRequest(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid json", `not json`},
		{"empty body", ``},
		{"missing url", `{}`},
		{"not a url", `{"url":"not-a-url"}`},
		{"relative path", `{"url":"/just/a/path"}`},
		{"javascript scheme", `{"url":"javascript:alert(1)"}`},
		{"ftp scheme", `{"url":"ftp://example.com/file"}`},
		{"negative expires_in", `{"url":"https://go.dev","expires_in":-1}`},
		{"expires_in over one year", `{"url":"https://go.dev","expires_in":31536001}`},
		{"expires_in not a number", `{"url":"https://go.dev","expires_in":"1h"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newTestMux(storage.NewMemory())

			rec := do(t, mux, "POST", "/shorten", tt.body)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
}

// collidingStorage pretends every code is already taken.
type collidingStorage struct {
	saves int
}

func (c *collidingStorage) Save(ctx context.Context, code string, link storage.Link) error {
	c.saves++
	return storage.ErrCodeExists
}

func (c *collidingStorage) Get(ctx context.Context, code string) (storage.Link, error) {
	return storage.Link{}, storage.ErrNotFound
}

func (c *collidingStorage) IncrementClicks(ctx context.Context, code string) error {
	return storage.ErrNotFound
}

func (c *collidingStorage) Clicks(ctx context.Context, code string) (int64, error) {
	return 0, storage.ErrNotFound
}

func TestShortenGivesUpAfterMaxAttempts(t *testing.T) {
	store := &collidingStorage{}
	mux := newTestMux(store)

	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if store.saves != maxSaveAttempts {
		t.Errorf("Save called %d times, want %d", store.saves, maxSaveAttempts)
	}
}

// failingStorage returns an unexpected error, like a database that is down.
type failingStorage struct{}

func (failingStorage) Save(ctx context.Context, code string, link storage.Link) error {
	return errors.New("database is down")
}

func (failingStorage) Get(ctx context.Context, code string) (storage.Link, error) {
	return storage.Link{}, errors.New("database is down")
}

func (failingStorage) IncrementClicks(ctx context.Context, code string) error {
	return errors.New("database is down")
}

func (failingStorage) Clicks(ctx context.Context, code string) (int64, error) {
	return 0, errors.New("database is down")
}

func TestShortenStorageError(t *testing.T) {
	mux := newTestMux(failingStorage{})

	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
