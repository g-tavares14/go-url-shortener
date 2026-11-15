package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

// newTestRedis connects to TEST_REDIS_URL and empties that database. Tests
// that need it are skipped when the variable is not set:
//
//	TEST_REDIS_URL=redis://localhost:6379/15 go test ./...
//
// Use a database number you don't use for anything else (15 above), because
// it gets flushed.
func newTestRedis(t *testing.T) *redis.Client {
	t.Helper()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL not set")
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatalf("invalid TEST_REDIS_URL: %v", err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { rdb.Close() })

	if err := rdb.FlushDB(t.Context()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
	return rdb
}

func TestCached(t *testing.T) {
	testStorage(t, func(t *testing.T) Storage {
		return NewCached(NewMemory(), newTestRedis(t), time.Hour)
	})
}

// countingStorage counts how often Get reaches the storage behind the cache.
type countingStorage struct {
	*Memory
	gets int
}

func (c *countingStorage) Get(ctx context.Context, code string) (Link, error) {
	c.gets++
	return c.Memory.Get(ctx, code)
}

func TestCachedServesFromRedis(t *testing.T) {
	behind := &countingStorage{Memory: NewMemory()}
	c := NewCached(behind, newTestRedis(t), time.Hour)
	if err := c.Save(t.Context(), "abc", Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}

	for range 3 {
		if _, err := c.Get(t.Context(), "abc"); err != nil {
			t.Fatal(err)
		}
	}

	// Save warmed the cache, so no Get should reach the storage behind.
	if behind.gets != 0 {
		t.Errorf("storage behind was read %d times, want 0", behind.gets)
	}
}

func TestCachedFillsCacheOnMiss(t *testing.T) {
	behind := &countingStorage{Memory: NewMemory()}
	// Saved straight into the storage behind, so the cache doesn't know it yet.
	if err := behind.Save(t.Context(), "abc", Link{URL: "https://go.dev"}); err != nil {
		t.Fatal(err)
	}
	c := NewCached(behind, newTestRedis(t), time.Hour)

	c.Get(t.Context(), "abc") // miss: reads behind and fills the cache
	c.Get(t.Context(), "abc") // hit

	if behind.gets != 1 {
		t.Errorf("storage behind was read %d times, want 1", behind.gets)
	}
}

func TestCachedTTLFollowsExpiry(t *testing.T) {
	rdb := newTestRedis(t)
	c := NewCached(NewMemory(), rdb, time.Hour)
	link := Link{URL: "https://go.dev", ExpiresAt: time.Now().Add(time.Minute)}
	if err := c.Save(t.Context(), "abc", link); err != nil {
		t.Fatal(err)
	}

	ttl, err := rdb.TTL(t.Context(), cacheKey("abc")).Result()
	if err != nil {
		t.Fatal(err)
	}
	// The link expires in a minute, so it must leave the cache by then,
	// not after the one-hour default.
	if ttl <= 0 || ttl > time.Minute {
		t.Errorf("cache TTL = %v, want at most 1m", ttl)
	}
}

func TestCachedWorksWhenRedisIsDown(t *testing.T) {
	// Nothing listens on this port, so every Redis call fails.
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:1", MaxRetries: -1, DialTimeout: 100 * time.Millisecond})
	t.Cleanup(func() { rdb.Close() })
	c := NewCached(NewMemory(), rdb, time.Hour)

	if err := c.Save(t.Context(), "abc", Link{URL: "https://go.dev"}); err != nil {
		t.Fatalf("Save with Redis down: %v", err)
	}
	got, err := c.Get(t.Context(), "abc")
	if err != nil {
		t.Fatalf("Get with Redis down: %v", err)
	}
	if got.URL != "https://go.dev" {
		t.Errorf("URL = %q, want %q", got.URL, "https://go.dev")
	}
}
