package wal_test

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func TestWALFlush(t *testing.T) {
	path := "test_wal_flush.log"
	w, err := wal.NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer os.Remove(path)
	defer w.Close()
	if err := w.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}
}
