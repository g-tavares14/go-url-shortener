package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"go-url-shortener/internal/storage"
)

const (
	maxSaveAttempts = 5
	// maxExpiresIn caps expires_in at one year, in seconds.
	maxExpiresIn = 365 * 24 * 60 * 60
)

type shortenRequest struct {
	URL string `json:"url"`
	// ExpiresIn is how many seconds the link stays valid. 0 or missing means forever.
	ExpiresIn int `json:"expires_in"`
}

type shortenResponse struct {
	Code      string     `json:"code"`
	ShortURL  string     `json:"short_url"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (s *Server) shorten(w http.ResponseWriter, r *http.Request) {
	var req shortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	if req.ExpiresIn < 0 || req.ExpiresIn > maxExpiresIn {
		http.Error(w, "invalid expires_in", http.StatusBadRequest)
		return
	}

	link := storage.Link{URL: req.URL}
	if req.ExpiresIn > 0 {
		link.ExpiresAt = time.Now().UTC().Add(time.Duration(req.ExpiresIn) * time.Second).Truncate(time.Second)
	}

	var code string

	for range maxSaveAttempts {
		code, err = generateCode()
		if err != nil {
			slog.Error("failed to generate code", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		err = s.Store.Save(r.Context(), code, link)
		if !errors.Is(err, storage.ErrCodeExists) {
			break
		}
	}
	if err != nil {
		slog.Error("failed to save url", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	resp := shortenResponse{Code: code, ShortURL: s.BaseURL + "/" + code}
	if !link.ExpiresAt.IsZero() {
		resp.ExpiresAt = &link.ExpiresAt
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to write shorten response", "error", err)
	}
}
