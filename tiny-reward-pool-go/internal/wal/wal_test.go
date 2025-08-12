package wal_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestParseWAL(t *testing.T) {
	path := "test_wal.log"
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create wal log: %v", err)
	}

	// Write test data in JSONL format
	encoder := json.NewEncoder(f)
	_ = encoder.Encode(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 1, ItemID: "gold", Success: true})
	_ = encoder.Encode(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 2, ItemID: "silver", Success: true})
	_ = encoder.Encode(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw, Error: types.ErrorPoolEmpty}, RequestID: 3, Success: false})
	_ = encoder.Encode(types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 4, ItemID: "bronze", Success: true})
	f.Close()

	items, err := wal.ParseWAL(path, walformatter.NewJSONFormatter())
	if err != nil {
		t.Fatalf("ParseWAL failed: %v", err)
	}
	defer os.Remove(path)

	if len(items) != 4 {
		t.Fatalf("expected 4 items, got %d", len(items))
	}

	expectedItem0 := types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 1, ItemID: "gold", Success: true}
	if items[0].Type != expectedItem0.Type || items[0].RequestID != expectedItem0.RequestID || items[0].ItemID != expectedItem0.ItemID || items[0].Success != expectedItem0.Success {
		t.Errorf("unexpected item 0: got %+v, want %+v", items[0], expectedItem0)
	}

	expectedItem2 := types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw, Error: types.ErrorPoolEmpty}, RequestID: 3, Success: false}
	if items[2].Type != expectedItem2.Type || items[2].RequestID != expectedItem2.RequestID || items[2].Success != expectedItem2.Success || items[2].Error != expectedItem2.Error {
		t.Errorf("unexpected item 2: got %+v, want %+v", items[2], expectedItem2)
	}
}

func TestLogDraw(t *testing.T) {
	path := "test_wal.log"
	fileStorage, err := walstorage.NewFileStorage(path)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	w, err := wal.NewWAL(path, walformatter.NewJSONFormatter(), fileStorage)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer os.Remove(path)
	defer w.Close()
	item := types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 1, ItemID: "gold", Success: true}
	if err := w.LogDraw(item); err != nil {
		t.Fatalf("LogDraw failed: %v", err)
	}
}

func TestWALFlush(t *testing.T) {
	path := "test_wal_flush.log"
	fileStorage, err := walstorage.NewFileStorage(path)
	if err != nil {
		t.Fatalf("Failed to create file storage: %v", err)
	}
	w, err := wal.NewWAL(path, walformatter.NewJSONFormatter(), fileStorage)
	if err != nil {
		t.Fatalf("Failed to create WAL: %v", err)
	}
	defer os.Remove(path)
	defer w.Close()

	// Log one item to flush
	item := types.WalLogDrawItem{WalLogItem: types.WalLogItem{Type: types.LogTypeDraw}, RequestID: 1, ItemID: "gold", Success: true}
	if err := w.LogDraw(item); err != nil {
		t.Fatalf("LogDraw failed: %v", err)
	}

	if err := w.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify content
	items, err := wal.ParseWAL(path, walformatter.NewJSONFormatter())
	if err != nil {
		t.Fatalf("ParseWAL failed after flush: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item after flush, got %d", len(items))
	}
	if items[0].ItemID != "gold" {
		t.Errorf("unexpected item after flush: %+v", items[0])
	}
}