package wal

import (
	"fmt"
	"os"
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

func (w *WAL) LogDraw(requestID uint64, itemID string, success bool) error {
	var line string
	if success {
		line = fmt.Sprintf("DRAW %d %s\n", requestID, itemID)
	} else {
		line = fmt.Sprintf("DRAW %d FAILED\n", requestID)
	}
	_, err := w.file.WriteString(line)
	return err
}

func (w *WAL) Close() error {
	return w.file.Close()
}
