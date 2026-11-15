package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"

	"go-url-shortener/internal/storage"
)

const maxSaveAttempts = 5

type shortenRequest struct {
	URL string `json:"url"`
}

type shortenResponse struct {
	Code     string `json:"code"`
	ShortURL string `json:"short_url"`
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

	var code string

	for range maxSaveAttempts {
		code, err = generateCode()
		if err != nil {
			slog.Error("failed to generate code", "error", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
		err = s.Store.Save(code, req.URL)
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to write shorten response", "error", err)
	}
}
