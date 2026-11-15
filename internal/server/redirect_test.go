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

// fakeClicks remembers which codes were recorded.
type fakeClicks struct {
	codes []string
}

func (f *fakeClicks) Record(code string) {
	f.codes = append(f.codes, code)
}

func TestRedirectRecordsClick(t *testing.T) {
	store := storage.NewMemory()
	if err := store.Save("abcdefgh", storage.Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	recorder := &fakeClicks{}
	s := &Server{Store: store, BaseURL: "http://test.local", Clicks: recorder}
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	do(t, mux, "GET", "/abcdefgh", "")

	if len(recorder.codes) != 1 || recorder.codes[0] != "abcdefgh" {
		t.Errorf("recorded clicks = %v, want [abcdefgh]", recorder.codes)
	}
}

func TestRedirectDoesNotRecordFailedVisits(t *testing.T) {
	store := storage.NewMemory()
	expired := storage.Link{URL: "https://go.dev", ExpiresAt: time.Now().Add(-time.Minute)}
	if err := store.Save("expired1", expired); err != nil {
		t.Fatal(err)
	}
	recorder := &fakeClicks{}
	s := &Server{Store: store, BaseURL: "http://test.local", Clicks: recorder}
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	do(t, mux, "GET", "/expired1", "")     // 410
	do(t, mux, "GET", "/doesnotexist", "") // 404

	if len(recorder.codes) != 0 {
		t.Errorf("recorded clicks = %v, want none", recorder.codes)
	}
}
