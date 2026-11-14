package server

import (
	"log/slog"
	"net/http"
)

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
		slog.Error("failed to write health response", "error", err)
	}
}
