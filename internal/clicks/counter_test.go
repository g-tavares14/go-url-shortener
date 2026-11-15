package clicks

import (
	"testing"
	"time"

	"go-url-shortener/internal/storage"
)

func TestCounterSavesQueuedClicksOnClose(t *testing.T) {
	store := storage.NewMemory()
	if err := store.Save("abc", storage.Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	c := NewCounter(store, 10)
	c.Start()

	for range 5 {
		c.Record("abc")
	}
	c.Close() // waits until every queued click is saved

	got, _ := store.Get("abc")
	if got.Clicks != 5 {
		t.Errorf("Clicks = %d, want 5", got.Clicks)
	}
}

// slowStorage blocks IncrementClicks until release is closed, to simulate a
// slow database.
type slowStorage struct {
	*storage.Memory
	entered chan struct{}
	release chan struct{}
}

func (s *slowStorage) IncrementClicks(code string) error {
	s.entered <- struct{}{}
	<-s.release
	return s.Memory.IncrementClicks(code)
}

func TestCounterRecordNeverBlocks(t *testing.T) {
	mem := storage.NewMemory()
	if err := mem.Save("abc", storage.Link{URL: "https://go.dev"}); err != nil {
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

	got, _ := mem.Get("abc")
	if got.Clicks != 2 {
		t.Errorf("Clicks = %d, want 2 (the third click is dropped)", got.Clicks)
	}
}
