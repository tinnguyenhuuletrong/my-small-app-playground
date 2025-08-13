package wal

import (
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type WAL struct {
	formatter types.LogFormatter
	storage   types.Storage
	buffer    []types.WalLogDrawItem
}

var _ types.WAL = (*WAL)(nil)

func (w *WAL) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	data, err := w.formatter.Encode(w.buffer)
	if err != nil {
		return err
	}

	if !w.storage.CanWrite(len(data)) {
		return types.ErrWALFull
	}

	err = w.storage.Write(data)
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
	return &WAL{formatter: format, storage: store, buffer: make([]types.WalLogDrawItem, 0, 4096)}, nil
}

func (w *WAL) LogDraw(item types.WalLogDrawItem) error {
	w.buffer = append(w.buffer, item)
	return nil
}

func (w *WAL) Close() error {
	return w.storage.Close()
}

func (w *WAL) Rotate(path string) error {
	return w.storage.Rotate(path)
}

// ParseWAL reads the WAL log file and returns a slice of WalLogDrawItem
func ParseWAL(path string, format types.LogFormatter) ([]types.WalLogDrawItem, error) {
	data, err := utils.ReadFileContent(path)
	if err != nil {
		return nil, err
	}

	return format.Decode(data)
}
