package storage

import (
	"context"
	"errors"
	"time"
)

// Link is what a short code points to. It never changes after it is saved,
// which is what makes it safe to cache.
type Link struct {
	URL string
	// ExpiresAt is when the link stops working. The zero value means it never expires.
	ExpiresAt time.Time
}

// Expired reports whether the link has expired at the given time.
func (l Link) Expired(now time.Time) bool {
	return !l.ExpiresAt.IsZero() && !now.Before(l.ExpiresAt)
}

// Storage keeps links and their click counts. Clicks live apart from Link
// because they change on every visit, while a Link never does.
type Storage interface {
	Save(ctx context.Context, code string, link Link) error
	Get(ctx context.Context, code string) (Link, error)
	IncrementClicks(ctx context.Context, code string) error
	Clicks(ctx context.Context, code string) (int64, error)
}

var (
	ErrNotFound   = errors.New("not found")
	ErrCodeExists = errors.New("code already exists")
)
