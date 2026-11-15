package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"go-url-shortener/internal/clicks"
	"go-url-shortener/internal/server"
	"go-url-shortener/internal/storage"
)

const (
	// clickBufferSize is how many clicks can wait to be saved before new ones are dropped.
	clickBufferSize = 1024
	// cacheTTL is how long a link stays in Redis without being read again.
	cacheTTL = time.Hour
	// startupTimeout limits how long connecting to Postgres and Redis may take.
	startupTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

// run holds the real main, so deferred Close calls run before os.Exit.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	port := 8080
	if v := os.Getenv("PORT"); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("invalid PORT %q: %w", v, err)
		}
		port = p
	}

	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("http://localhost:%d", port)
	}
	baseURL = strings.TrimSuffix(baseURL, "/")

	store, closeStore, err := newStorage(ctx)
	if err != nil {
		return err
	}
	defer closeStore()

	counter := clicks.NewCounter(store, clickBufferSize)
	counter.Start()

	srv := &server.Server{
		Store:   store,
		BaseURL: baseURL,
		Clicks:  counter,
	}

	err = srv.Run(ctx, port)

	// Run only returns after in-flight requests finish, so no more clicks
	// arrive. Close saves the ones still queued, before the storage closes.
	counter.Close()

	return err
}

// newStorage picks the storage from the environment:
//   - DATABASE_URL set: Postgres. Otherwise: in memory, lost on restart.
//   - REDIS_URL set: a Redis cache in front of it.
//
// The returned function closes every connection it opened.
func newStorage(ctx context.Context) (storage.Storage, func(), error) {
	ctx, cancel := context.WithTimeout(ctx, startupTimeout)
	defer cancel()

	var store storage.Storage
	var closers []func()
	closeAll := func() {
		for i := len(closers) - 1; i >= 0; i-- {
			closers[i]()
		}
	}

	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		pg, err := storage.NewPostgres(ctx, databaseURL)
		if err != nil {
			return nil, nil, err
		}
		closers = append(closers, pg.Close)
		store = pg
		slog.Info("using postgres storage")
	} else {
		store = storage.NewMemory()
		slog.Warn("DATABASE_URL not set, using in-memory storage: links are lost on restart")
	}

	if redisURL := os.Getenv("REDIS_URL"); redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			closeAll()
			return nil, nil, fmt.Errorf("invalid REDIS_URL: %w", err)
		}
		rdb := redis.NewClient(opts)
		if err := rdb.Ping(ctx).Err(); err != nil {
			rdb.Close()
			closeAll()
			return nil, nil, fmt.Errorf("ping redis: %w", err)
		}
		closers = append(closers, func() { rdb.Close() })
		store = storage.NewCached(store, rdb, cacheTTL)
		slog.Info("using redis cache", "ttl", cacheTTL)
	}

	return store, closeAll, nil
}
