package server

import (
	"encoding/json"
	"errors"
	"go-url-shortener/internal/storage"
	"log/slog"
	"net/http"
	"net/url"
)

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
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	u, err := url.ParseRequestURI(req.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	var code string

	for range 5 {
		code, err = generateCode()
		if err != nil {
			slog.Error("Failed to generate code", slog.Any("err", err))
			http.Error(w, "Failed to generate code", http.StatusInternalServerError)
			return
		}
		err = s.Store.Save(code, req.URL)
		if !errors.Is(err, storage.ErrCodeExists) {
			break
		}
	}
	if err != nil {
		slog.Error("Failed to save code", slog.Any("err", err))
		http.Error(w, "Failed to save code", http.StatusInternalServerError)
		return
	}

	resp := shortenResponse{Code: code, ShortURL: "http://localhost:8080/" + code}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("failed to write shorten response", "error", err)
	}
}
