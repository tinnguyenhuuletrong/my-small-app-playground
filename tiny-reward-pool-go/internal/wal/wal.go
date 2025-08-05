package wal

import (
	"bufio"
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type WAL struct {
	file *os.File
}

func (w *WAL) Flush() error {
	return w.file.Sync()
}

func NewWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{file: f}, nil
}

func (w *WAL) LogDraw(item types.WalLogItem) error {
	var line string
	if item.Success {
		line = fmt.Sprintf("DRAW %d %s\n", item.RequestID, item.ItemID)
	} else {
		line = fmt.Sprintf("DRAW %d FAILED\n", item.RequestID)
	}
	_, err := w.file.WriteString(line)
	return err
}

func (w *WAL) Close() error {
	return w.file.Close()
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
