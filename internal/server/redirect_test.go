package server

import (
	"net/http"
	"testing"
	"time"

	"go-url-shortener/internal/storage"
)

func TestRedirectExistingCode(t *testing.T) {
	store := storage.NewMemory()
	if err := store.Save("abcdefgh", storage.Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	mux := newTestMux(store)

	rec := do(t, mux, "GET", "/abcdefgh", "")

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "https://go.dev" {
		t.Errorf("Location = %q, want %q", loc, "https://go.dev")
	}
}

func TestRedirectNotYetExpired(t *testing.T) {
	store := storage.NewMemory()
	link := storage.Link{URL: "https://go.dev", ExpiresAt: time.Now().Add(time.Hour)}
	if err := store.Save("abcdefgh", link); err != nil {
		t.Fatal(err)
	}
	mux := newTestMux(store)

	rec := do(t, mux, "GET", "/abcdefgh", "")

	if rec.Code != http.StatusFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusFound)
	}
}

func TestRedirectExpired(t *testing.T) {
	store := storage.NewMemory()
	link := storage.Link{URL: "https://go.dev", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := store.Save("abcdefgh", link); err != nil {
		t.Fatal(err)
	}
	mux := newTestMux(store)

	rec := do(t, mux, "GET", "/abcdefgh", "")

	if rec.Code != http.StatusGone {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusGone)
	}
	if loc := rec.Header().Get("Location"); loc != "" {
		t.Errorf("Location = %q, want no redirect", loc)
	}
}

func TestRedirectUnknownCode(t *testing.T) {
	mux := newTestMux(storage.NewMemory())

	rec := do(t, mux, "GET", "/doesnotexist", "")

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRedirectStorageError(t *testing.T) {
	mux := newTestMux(failingStorage{})

	rec := do(t, mux, "GET", "/abcdefgh", "")

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
}

func TestShortenThenRedirect(t *testing.T) {
	mux := newTestMux(storage.NewMemory())

	rec := do(t, mux, "POST", "/shorten", `{"url":"https://go.dev/doc"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("shorten status = %d, want %d", rec.Code, http.StatusCreated)
	}
	code := decodeCode(t, rec.Body.Bytes())

	rec = do(t, mux, "GET", "/"+code, "")

	if rec.Code != http.StatusFound {
		t.Fatalf("redirect status = %d, want %d", rec.Code, http.StatusFound)
	}
	if loc := rec.Header().Get("Location"); loc != "https://go.dev/doc" {
		t.Errorf("Location = %q, want %q", loc, "https://go.dev/doc")
	}
}

func TestHealthStillWorks(t *testing.T) {
	// /healthz must not be captured by the GET /{code} route.
	mux := newTestMux(storage.NewMemory())

	rec := do(t, mux, "GET", "/healthz", "")

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
}
