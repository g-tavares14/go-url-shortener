package server

import (
	"errors"
	"go-url-shortener/internal/storage"
	"log/slog"
	"net/http"
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
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, target, http.StatusFound)
}
