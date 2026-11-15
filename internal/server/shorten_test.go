package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

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
	if want := "http://localhost:8080/" + resp.Code; resp.ShortURL != want {
		t.Errorf("short_url = %q, want %q", resp.ShortURL, want)
	}

	got, err := store.Get(resp.Code)
	if err != nil {
		t.Fatalf("code %q was not saved: %v", resp.Code, err)
	}
	if got != "https://go.dev" {
		t.Errorf("saved url = %q, want %q", got, "https://go.dev")
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

func (c *collidingStorage) Save(code, url string) error {
	c.saves++
	return storage.ErrCodeExists
}

func (c *collidingStorage) Get(code string) (string, error) {
	return "", storage.ErrNotFound
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

func (failingStorage) Save(code, url string) error {
	return errors.New("database is down")
}

func (failingStorage) Get(code string) (string, error) {
	return "", errors.New("database is down")
}

func TestShortenStorageError(t *testing.T) {
	mux := newTestMux(failingStorage{})

	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev"}`)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}
