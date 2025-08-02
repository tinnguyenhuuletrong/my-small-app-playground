package wal_test

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func TestLogDraw(t *testing.T) {
	path := "test_wal.log"
	w, err := wal.NewWAL(path)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer os.Remove(path)
	defer w.Close()
	item := types.WalLogItem{RequestID: 1, ItemID: "gold", Success: true}
	if err := w.LogDraw(item); err != nil {
		t.Fatalf("LogDraw failed: %v", err)
	}
}
