package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-url-shortener/internal/storage"
)

// newTestMux builds the same routes as Run, backed by the given storage,
// without opening a port.
func newTestMux(store storage.Storage) *http.ServeMux {
	s := &Server{Store: store, BaseURL: "http://test.local"}
	mux := http.NewServeMux()
	s.registerRoutes(mux)
	return mux
}

// do sends a request to the mux and returns the recorded response.
func do(t *testing.T, mux *http.ServeMux, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

// decodeCode reads the code field from a /shorten response body.
func decodeCode(t *testing.T, body []byte) string {
	t.Helper()
	var resp shortenResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return resp.Code
}
