package storage

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

// uniqueViolation is the Postgres error code for a duplicate primary key.
const uniqueViolation = "23505"

// Postgres stores links in the links table. It is the source of truth.
type Postgres struct {
	pool *pgxpool.Pool
}

// NewPostgres connects to the database and creates the links table if it
// does not exist yet. Call Close when done.
func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() {
	p.pool.Close()
}

func (p *Postgres) Save(ctx context.Context, code string, link Link) error {
	_, err := p.pool.Exec(ctx,
		`INSERT INTO links (code, url, expires_at) VALUES ($1, $2, $3)`,
		code, link.URL, nullTime(link.ExpiresAt),
	)

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
		return ErrCodeExists
	}
	if err != nil {
		return fmt.Errorf("insert link: %w", err)
	}

	return nil
}

func (p *Postgres) Get(ctx context.Context, code string) (Link, error) {
	var link Link
	var expiresAt *time.Time

	err := p.pool.QueryRow(ctx,
		`SELECT url, expires_at FROM links WHERE code = $1`,
		code,
	).Scan(&link.URL, &expiresAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Link{}, ErrNotFound
	}
	if err != nil {
		return Link{}, fmt.Errorf("select link: %w", err)
	}

	if expiresAt != nil {
		link.ExpiresAt = expiresAt.UTC()
	}
	return link, nil
}

func (p *Postgres) IncrementClicks(ctx context.Context, code string) error {
	tag, err := p.pool.Exec(ctx,
		`UPDATE links SET clicks = clicks + 1 WHERE code = $1`,
		code,
	)
	if err != nil {
		return fmt.Errorf("increment clicks: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (p *Postgres) Clicks(ctx context.Context, code string) (int64, error) {
	var clicks int64

	err := p.pool.QueryRow(ctx,
		`SELECT clicks FROM links WHERE code = $1`,
		code,
	).Scan(&clicks)

	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("select clicks: %w", err)
	}

	return clicks, nil
}

// nullTime turns the zero time into SQL NULL, meaning "never expires".
func nullTime(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
