package wal_test

import (
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func TestParseWAL(t *testing.T) {
	path := "test_wal.log"
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create wal log: %v", err)
	}
	_, _ = f.WriteString("DRAW 1 gold\nDRAW 2 silver\nDRAW 3 FAILED\nDRAW 4 bronze\n")
	f.Close()

	items, err := wal.ParseWAL(path)
	if err != nil {
		t.Fatalf("ParseWAL failed: %v", err)
	}
	if len(items) != 4 {
		t.Errorf("expected 4 items, got %d", len(items))
	}
	if items[0] != (types.WalLogItem{RequestID: 1, ItemID: "gold", Success: true}) {
		t.Errorf("unexpected item: %+v", items[0])
	}
	if items[2].Success != false || items[2].ItemID != "" {
		t.Errorf("expected failed log for item 3, got %+v", items[2])
	}
	os.Remove(path)
}

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
