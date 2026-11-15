package server

import "net/http"

func (s *Server) registerRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /shorten", s.shorten)
	mux.HandleFunc("GET /{code}", s.redirect)
	mux.HandleFunc("GET /healthz", s.health)
}
