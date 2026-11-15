package storage

import "testing"

func TestMemory(t *testing.T) {
	testStorage(t, func(t *testing.T) Storage {
		return NewMemory()
	})
}
