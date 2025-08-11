package wal

import (
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type WAL struct {
	formatter types.LogFormatter
	storage   types.Storage
	buffer    [][]byte // Now stores pre-encoded data
}

var _ types.WAL = (*WAL)(nil)

func (w *WAL) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	err := w.storage.WriteAll(w.buffer)
	if err != nil {
		return err
	}

	w.buffer = w.buffer[:0]
	return w.storage.Flush()
}

func NewWAL(path string, format types.LogFormatter, store types.Storage) (*WAL, error) {
	if format == nil {
		format = formatter.NewJSONFormatter()
	}
	if store == nil {
		var err error
		store, err = storage.NewFileStorage(path)
		if err != nil {
			return nil, err
		}
	}

	// Preallocate buffer for performance (e.g., 4096 entries)
	return &WAL{formatter: format, storage: store, buffer: make([][]byte, 0, 4096)}, nil
}

func (w *WAL) LogDraw(item types.WalLogDrawItem) error {
	encodedItem, err := w.formatter.Encode([]types.WalLogDrawItem{item})
	if err != nil {
		return err
	}
	w.buffer = append(w.buffer, encodedItem)
	return nil
}

func (w *WAL) Close() error {
	return w.storage.Close()
}

func (w *WAL) Rotate(path string) error {
	if len(w.buffer) > 0 {
		return types.ErrWalBufferNotEmpty
	}

	return w.storage.Rotate(path)
}

// ParseWAL reads the WAL log file and returns a slice of WalLogDrawItem
func ParseWAL(path string, format types.LogFormatter, store types.Storage) ([]types.WalLogDrawItem, error) {
	lines, err := store.ReadAll()
	if err != nil {
		return nil, err
	}

	var allData []byte
	for _, line := range lines {
		allData = append(allData, line...)
		allData = append(allData, '\n') // Add newline back for proper decoding
	}

	return format.Decode(allData)
}
