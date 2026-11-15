package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"go-url-shortener/internal/storage"
)

type statsResponse struct {
	Code      string     `json:"code"`
	URL       string     `json:"url"`
	Clicks    int64      `json:"clicks"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	link, err := s.Store.Get(code)

	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		slog.Error("failed to get stats", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := statsResponse{Code: code, URL: link.URL, Clicks: link.Clicks}
	if !link.ExpiresAt.IsZero() {
		resp.ExpiresAt = &link.ExpiresAt
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to write stats response", "error", err)
	}
}
