package storage

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cached puts Redis in front of another Storage (cache-aside).
//
// Only links are cached: they never change after they are saved, so the
// cache can never be stale. Clicks change on every visit, so they always go
// straight to the storage behind.
//
// If Redis fails, Cached logs it and falls back to the storage behind: the
// cache makes things faster, it is never required.
type Cached struct {
	next Storage
	rdb  *redis.Client
	ttl  time.Duration
}

// NewCached wraps next with a Redis cache. Entries live for at most ttl, and
// never past the link's own expiry.
func NewCached(next Storage, rdb *redis.Client, ttl time.Duration) *Cached {
	return &Cached{next: next, rdb: rdb, ttl: ttl}
}

type cachedLink struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func cacheKey(code string) string {
	return "link:" + code
}

func (c *Cached) Save(ctx context.Context, code string, link Link) error {
	if err := c.next.Save(ctx, code, link); err != nil {
		return err
	}

	// A new link is usually opened right after it is created, so warm the cache.
	c.store(ctx, code, link)
	return nil
}

func (c *Cached) Get(ctx context.Context, code string) (Link, error) {
	data, err := c.rdb.Get(ctx, cacheKey(code)).Bytes()
	if err == nil {
		var cl cachedLink
		if err := json.Unmarshal(data, &cl); err == nil {
			return Link{URL: cl.URL, ExpiresAt: cl.ExpiresAt}, nil
		}
		slog.Warn("invalid cache entry, ignoring it", "code", code)
	} else if !errors.Is(err, redis.Nil) {
		// redis.Nil only means "not in the cache". Anything else is Redis failing.
		slog.Warn("cache read failed, using storage", "code", code, "error", err)
	}

	link, err := c.next.Get(ctx, code)
	if err != nil {
		return Link{}, err
	}

	c.store(ctx, code, link)
	return link, nil
}

func (c *Cached) IncrementClicks(ctx context.Context, code string) error {
	return c.next.IncrementClicks(ctx, code)
}

func (c *Cached) Clicks(ctx context.Context, code string) (int64, error) {
	return c.next.Clicks(ctx, code)
}

// store writes the link to Redis. Errors are only logged.
func (c *Cached) store(ctx context.Context, code string, link Link) {
	ttl := c.ttl
	if !link.ExpiresAt.IsZero() {
		untilExpiry := time.Until(link.ExpiresAt)
		if untilExpiry <= 0 {
			return // already expired: nothing worth caching
		}
		ttl = min(ttl, untilExpiry)
	}

	data, err := json.Marshal(cachedLink{URL: link.URL, ExpiresAt: link.ExpiresAt})
	if err != nil {
		slog.Warn("failed to encode cache entry", "code", code, "error", err)
		return
	}

	if err := c.rdb.Set(ctx, cacheKey(code), data, ttl).Err(); err != nil {
		slog.Warn("cache write failed", "code", code, "error", err)
	}
}
