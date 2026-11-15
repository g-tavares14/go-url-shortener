package storage

import (
	"os"
	"testing"
)

// TestPostgres needs a real database. It is skipped unless TEST_DATABASE_URL
// is set, for example with `docker compose up -d`:
//
//	TEST_DATABASE_URL=postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable go test ./...
//
// It empties the links table, so never point it at data you want to keep.
func TestPostgres(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	testStorage(t, func(t *testing.T) Storage {
		pg, err := NewPostgres(t.Context(), databaseURL)
		if err != nil {
			t.Fatalf("NewPostgres: %v", err)
		}
		t.Cleanup(pg.Close)

		if _, err := pg.pool.Exec(t.Context(), "TRUNCATE links"); err != nil {
			t.Fatalf("truncate: %v", err)
		}
		return pg
	})
}
