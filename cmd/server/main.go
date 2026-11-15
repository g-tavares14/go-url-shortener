package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"

	"go-url-shortener/internal/clicks"
	"go-url-shortener/internal/server"
	"go-url-shortener/internal/storage"
)

// clickBufferSize is how many clicks can wait to be saved before new ones are dropped.
const clickBufferSize = 1024

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

	store := storage.NewMemory()

	counter := clicks.NewCounter(store, clickBufferSize)
	counter.Start()

	srv := &server.Server{
		Store:   store,
		BaseURL: baseURL,
		Clicks:  counter,
	}

	err := srv.Run(ctx, port)

	// Run only returns after in-flight requests finish, so no more clicks
	// arrive. Close saves the ones still queued.
	counter.Close()

	if err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
