package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"go-url-shortener/internal/storage"
)

// ClickRecorder records that a short link was opened.
type ClickRecorder interface {
	Record(code string)
}

type Server struct {
	Store   storage.Storage
	BaseURL string
	// Clicks is optional: when nil, clicks are not counted.
	Clicks ClickRecorder
}

func (s *Server) Run(ctx context.Context, port int) error {
	mux := http.NewServeMux()
	s.registerRoutes(mux)

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- httpServer.ListenAndServe()
	}()

	slog.Info("server is running", "port", port)

	select {
	case err := <-serveErr:
		return err
	case <-ctx.Done():
		slog.Info("shutting down server")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
