package server

import (
	"errors"
	"log/slog"
	"net/http"

	"go-url-shortener/internal/storage"
)

func (s *Server) redirect(w http.ResponseWriter, r *http.Request) {
	code := r.PathValue("code")
	target, err := s.Store.Get(code)

	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		slog.Error("failed to get redirect", "error", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
