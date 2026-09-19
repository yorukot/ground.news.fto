// Package dbtest gives tests a migrated, empty Postgres database.
//
// Tests that need it call Open, which skips the test unless
// TEST_DATABASE_URL is set (`make test-db` creates a database for it). It
// must never point at a database whose data matters: every table is emptied.
package dbtest

import (
	"context"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/yorukot/ground-news-tw/internal/db"
)

var migrateOnce sync.Once

func Open(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()

	var migrateErr error
	migrateOnce.Do(func() { migrateErr = db.Migrate(ctx, url) })
	if migrateErr != nil {
		t.Fatalf("migrate test database: %v", migrateErr)
	}

	pool, err := db.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `TRUNCATE outlets, events, articles, summaries, entities, event_entities, article_entities RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("empty test database: %v", err)
	}
	return pool
}
