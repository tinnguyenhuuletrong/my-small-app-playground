package wal

import (
	"bufio"
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type WAL struct {
	file   *os.File
	buffer []types.WalLogItem
}

var _ types.WAL = (*WAL)(nil)

func (w *WAL) Flush() error {
	if len(w.buffer) == 0 {
		return nil
	}
	var allLines []byte
	for _, item := range w.buffer {
		var line string
		if item.Success {
			line = fmt.Sprintf("DRAW %d %s\n", item.RequestID, item.ItemID)
		} else {
			line = fmt.Sprintf("DRAW %d FAILED\n", item.RequestID)
		}
		allLines = append(allLines, []byte(line)...)
	}
	if len(allLines) > 0 {
		if _, err := w.file.Write(allLines); err != nil {
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
	return &WAL{file: f, buffer: make([]types.WalLogItem, 0, 4096)}, nil
}

func (w *WAL) LogDraw(item types.WalLogItem) error {
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

// ParseWAL reads the WAL log file and returns a slice of WalLogItem
func ParseWAL(path string) ([]types.WalLogItem, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var items []types.WalLogItem
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		var reqID uint64
		var itemID string
		n, _ := fmt.Sscanf(line, "DRAW %d %s", &reqID, &itemID)
		if n == 2 {
			if itemID == "FAILED" {
				items = append(items, types.WalLogItem{RequestID: reqID, ItemID: "", Success: false})
			} else {
				items = append(items, types.WalLogItem{RequestID: reqID, ItemID: itemID, Success: true})
			}
		}
	}
	return items, scanner.Err()
}
