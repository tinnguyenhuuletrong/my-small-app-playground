package wal

import (
	"bufio"
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type WAL struct {
	file   *os.File
	buffer []types.WalLogDrawItem
}

var _ types.WAL = (*WAL)(nil)

func (w *WAL) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}

	encoder := json.NewEncoder(w.file)
	for _, item := range w.buffer {
		if err := encoder.Encode(item); err != nil {
			return err
		}
	}

	w.buffer = w.buffer[:0]
	return w.file.Sync()
}

func NewWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	// Preallocate buffer for performance (e.g., 4096 entries)
	return &WAL{file: f, buffer: make([]types.WalLogDrawItem, 0, 4096)}, nil
}

func (w *WAL) LogDraw(item types.WalLogDrawItem) error {
	w.buffer = append(w.buffer, item)
	return nil
}

func (w *WAL) Close() error {
	return w.file.Close()
}

func (w *WAL) Rotate(path string) error {
	if len(w.buffer) > 0 {
		return types.ErrWalBufferNotEmpty
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

// ParseWAL reads the WAL log file and returns a slice of WalLogDrawItem
func ParseWAL(path string) ([]types.WalLogDrawItem, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty slice if WAL file doesn't exist
		}
		return nil, err
	}
	defer f.Close()

	var items []types.WalLogDrawItem
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var item types.WalLogDrawItem
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	return items, scanner.Err()
}
