package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"go-url-shortener/internal/storage"
)

func TestStats(t *testing.T) {
	store := storage.NewMemory()
	expiresAt := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := store.Save("abcdefgh", storage.Link{URL: "https://go.dev", ExpiresAt: expiresAt}); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := store.IncrementClicks("abcdefgh"); err != nil {
			t.Fatal(err)
		}
	}
	mux := newTestMux(store)

	rec := do(t, mux, "GET", "/abcdefgh/stats", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var resp statsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Code != "abcdefgh" || resp.URL != "https://go.dev" || resp.Clicks != 2 {
		t.Errorf("stats = %+v, want code abcdefgh, url https://go.dev, clicks 2", resp)
	}
	if resp.ExpiresAt == nil || !resp.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expires_at = %v, want %v", resp.ExpiresAt, expiresAt)
	}
}

func TestStatsUnknownCode(t *testing.T) {
	mux := newTestMux(storage.NewMemory())

	rec := do(t, mux, "GET", "/doesnotexist/stats", "")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
