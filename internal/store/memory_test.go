package store_test

import (
	"testing"

	"habit/internal/store"
	"habit/internal/store/storetest"
)

func TestMemory(t *testing.T) {
	storetest.Run(t, func(*testing.T) store.Store { return store.NewMemory() })
}
