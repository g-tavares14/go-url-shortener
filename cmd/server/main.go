package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"go-url-shortener/internal/server"
	"go-url-shortener/internal/storage"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := &server.Server{
		Store: storage.NewMemory(),
	}

	if err := srv.Run(ctx, 8080); err != nil {
		log.Fatal(err)
	}
}
