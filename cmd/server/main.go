package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"go-url-shortener/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	srv := &server.Server{}

	if err := srv.Run(ctx, 8080); err != nil {
		log.Fatal(err)
	}
}
