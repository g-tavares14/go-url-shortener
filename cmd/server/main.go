package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"go-url-shortener/internal/server"
	"go-url-shortener/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			slog.Error("invalid PORT", "value", v, "error", err)
			os.Exit(1)
		}
		port = p
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	srv := &server.Server{
		Store:   storage.NewMemory(),
		BaseURL: baseURL,
	}

	if err := srv.Run(ctx, port); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
