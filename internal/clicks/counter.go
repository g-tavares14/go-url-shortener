// Package clicks counts link clicks in the background, so a redirect never
// waits for the storage to record a click.
package clicks

import (
	"context"
	"log/slog"
	"time"

	"go-url-shortener/internal/storage"
)

// Counter receives clicks on a buffered channel and a single goroutine saves
// them to the storage.
type Counter struct {
	store  storage.Storage
	events chan string
	done   chan struct{}
}

// NewCounter creates a Counter that holds up to bufferSize pending clicks.
// Call Start before Record, and Close when no more clicks will be recorded.
func NewCounter(store storage.Storage, bufferSize int) *Counter {
	return &Counter{
		store:  store,
		events: make(chan string, bufferSize),
		done:   make(chan struct{}),
	}
}

// Start launches the goroutine that saves clicks.
func (c *Counter) Start() {
	go c.run()
}

// Record queues a click without blocking. If the queue is full, the click is
// dropped: losing a count is better than slowing down a redirect.
func (c *Counter) Record(code string) {
	select {
	case c.events <- code:
	default:
		slog.Warn("click dropped, queue is full", "code", code)
	}
}

// Close stops accepting clicks and waits until the queued ones are saved.
// Record must not be called after Close.
func (c *Counter) Close() {
	close(c.events)
	<-c.done
}

func (c *Counter) run() {
	defer close(c.done)

	for code := range c.events {
		c.save(code)
	}
}

// saveTimeout limits how long one click may wait for the storage, so a stuck
// database cannot stall the queue forever.
const saveTimeout = 5 * time.Second

func (c *Counter) save(code string) {
	// Clicks outlive the request that caused them, so they get their own
	// context instead of the request's.
	ctx, cancel := context.WithTimeout(context.Background(), saveTimeout)
	defer cancel()

	if err := c.store.IncrementClicks(ctx, code); err != nil {
		slog.Error("failed to save click", "code", code, "error", err)
	}
}
