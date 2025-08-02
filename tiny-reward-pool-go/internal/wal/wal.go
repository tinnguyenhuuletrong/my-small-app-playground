package wal

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type WAL struct {
	file *os.File
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
