package clicks

import (
	"context"
	"testing"
	"time"

	"go-url-shortener/internal/storage"
)

func TestCounterSavesQueuedClicksOnClose(t *testing.T) {
	store := storage.NewMemory()
	if err := store.Save(t.Context(), "abc", storage.Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	c := NewCounter(store, 10)
	c.Start()

	for range 5 {
		c.Record("abc")
	}
	c.Close() // waits until every queued click is saved

	if got, _ := store.Clicks(t.Context(), "abc"); got != 5 {
		t.Errorf("Clicks = %d, want 5", got)
	}
}

// slowStorage blocks IncrementClicks until release is closed, to simulate a
// slow database.
type slowStorage struct {
	*storage.Memory
	entered chan struct{}
	release chan struct{}
}

func (s *slowStorage) IncrementClicks(ctx context.Context, code string) error {
	s.entered <- struct{}{}
	<-s.release
	return s.Memory.IncrementClicks(ctx, code)
}

func TestCounterRecordNeverBlocks(t *testing.T) {
	mem := storage.NewMemory()
	if err := mem.Save(t.Context(), "abc", storage.Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	store := &slowStorage{Memory: mem, entered: make(chan struct{}, 10), release: make(chan struct{})}
	c := NewCounter(store, 1)
	c.Start()

	c.Record("abc") // taken by the goroutine, which now blocks in the storage
	<-store.entered
	c.Record("abc") // fills the buffer of 1

	// The queue is full and the storage is stuck. Record must still return
	// right away, dropping the click instead of slowing down the redirect.
	returned := make(chan struct{})
	go func() {
		c.Record("abc")
		close(returned)
	}()
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("Record blocked while the queue was full")
	}

	close(store.release)
	c.Close()

	if got, _ := mem.Clicks(t.Context(), "abc"); got != 2 {
		t.Errorf("Clicks = %d, want 2 (the third click is dropped)", got)
	}
}
