package wal_test

import (
	
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestWAL_JSON(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")

	// Create a new WAL with JSON formatter
	w, err := wal.NewWAL(walPath, formatter.NewJSONFormatter(), nil)
	require.NoError(t, err)

	// Log some entries
	drawItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       1,
		ItemID:          "item1",
		Success:         true,
	}
	updateItem := types.WalLogUpdateItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
		ItemID:          "item2",
		Quantity:        10,
		Probability:     100,
	}
	w.LogDraw(drawItem)
	w.LogUpdate(updateItem)

	// Flush and close the WAL
	err = w.Flush()
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)

	// Parse the WAL file
	entries, hdr, err := wal.ParseWAL(walPath, formatter.NewJSONFormatter())
	require.NoError(t, err)
	require.NotNil(t, hdr)
	assert.Len(t, entries, 2)
	assert.Equal(t, types.WALStatusClosed, hdr.Status)

	// Check the first entry
	parsedDrawItem, ok := entries[0].(*types.WalLogDrawItem)
	require.True(t, ok)
	assert.Equal(t, drawItem.RequestID, parsedDrawItem.RequestID)
	assert.Equal(t, drawItem.ItemID, parsedDrawItem.ItemID)
	assert.Equal(t, drawItem.Success, parsedDrawItem.Success)

	// Check the second entry
	parsedUpdateItem, ok := entries[1].(*types.WalLogUpdateItem)
	require.True(t, ok)
	assert.Equal(t, updateItem.ItemID, parsedUpdateItem.ItemID)
	assert.Equal(t, updateItem.Quantity, parsedUpdateItem.Quantity)
	assert.Equal(t, updateItem.Probability, parsedUpdateItem.Probability)
}

func TestWAL_StringLine(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")

	// Create a new WAL with StringLine formatter
	w, err := wal.NewWAL(walPath, formatter.NewStringLineFormatter(), nil)
	require.NoError(t, err)

	// Log some entries
	drawItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       1,
		ItemID:          "item1",
		Success:         true,
	}
	w.LogDraw(drawItem)

	// Flush and close
	err = w.Flush()
	require.NoError(t, err)
	err = w.Close()
	require.NoError(t, err)

	// Parse the WAL file
	entries, _, err := wal.ParseWAL(walPath, formatter.NewStringLineFormatter())
	require.NoError(t, err)
	assert.Len(t, entries, 1)

	// Check the first entry
	parsedDrawItem, ok := entries[0].(*types.WalLogDrawItem)
	require.True(t, ok)
	assert.Equal(t, drawItem.RequestID, parsedDrawItem.RequestID)
}

func TestWAL_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")
	archivePath := filepath.Join(tempDir, "test.wal.rotated")

	// Create a new WAL
	w, err := wal.NewWAL(walPath, formatter.NewJSONFormatter(), nil)
	require.NoError(t, err)

	// Log an entry and flush
	drawItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       1,
		ItemID:          "item1",
		Success:         true,
	}
	w.LogDraw(drawItem)
	err = w.Flush()
	require.NoError(t, err)

	// Rotate the WAL
	err = w.Rotate(archivePath)
	require.NoError(t, err)

	// Check that the archived WAL exists and has a closed status
	_, hdr, err := wal.ParseWAL(archivePath, formatter.NewJSONFormatter())
	require.NoError(t, err)
	require.NotNil(t, hdr)
	assert.Equal(t, types.WALStatusClosed, hdr.Status)
	assert.Contains(t, string(hdr.NextWALPath[:]), archivePath)

	// Log another entry to the new WAL
	updateItem := types.WalLogUpdateItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeUpdate},
		ItemID:          "item2",
		Quantity:        10,
		Probability:     100,
	}
	w.LogUpdate(updateItem)
	err = w.Flush()
	require.NoError(t, err)
	w.Close()

	// Verify the content of the new WAL
	entries, hdr, err := wal.ParseWAL(walPath, formatter.NewJSONFormatter())
	require.NoError(t, err)
	require.NotNil(t, hdr)
	assert.Len(t, entries, 1)
	assert.Equal(t, types.WALStatusClosed, hdr.Status)
	parsedUpdateItem, ok := entries[0].(*types.WalLogUpdateItem)
	require.True(t, ok)
	assert.Equal(t, "item2", parsedUpdateItem.ItemID)
}

func TestWAL_Full(t *testing.T) {
	tempDir := t.TempDir()
	walPath := filepath.Join(tempDir, "test.wal")

	// Create a file storage with a small capacity
	// Capacity needs to be larger than header size
	storage, err := storage.NewFileStorage(walPath, storage.FileStorageOpt{SizeFileInBytes: types.WALHeaderSize + 10})
	require.NoError(t, err)

	// Create a new WAL
	w, err := wal.NewWAL(walPath, formatter.NewJSONFormatter(), storage)
	require.NoError(t, err)

	// Log an entry that will exceed the capacity
	drawItem := types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       1,
		ItemID:          "a-very-long-item-id-to-exceed-capacity",
		Success:         true,
	}
	w.LogDraw(drawItem)

	// Flush should return ErrWALFull
	err = w.Flush()
	assert.Equal(t, types.ErrWALFull, err)
}
