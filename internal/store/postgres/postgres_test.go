package postgres

import (
	"context"
	"os"
	"testing"

	"habit/internal/store"
	"habit/internal/store/storetest"
)

// Set HABIT_TEST_DATABASE_URL to a throwaway database to run these tests.
// They empty every Habit table in it.
func TestPostgres(t *testing.T) {
	url := os.Getenv("HABIT_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("HABIT_TEST_DATABASE_URL not set")
	}
	storetest.Run(t, func(t *testing.T) store.Store {
		ctx := context.Background()
		s, err := Open(ctx, url)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.pool.Exec(ctx, `TRUNCATE accounts, sessions, messages, threads, replies, notifications, thread_votes, settings`); err != nil {
			t.Fatal(err)
		}
		return s
	})
}
