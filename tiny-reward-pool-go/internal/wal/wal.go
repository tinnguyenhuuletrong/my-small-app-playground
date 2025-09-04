package wal

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

type WAL struct {
	formatter types.LogFormatter
	storage   types.Storage
	buffer    []types.WalLogEntry
}

var _ types.WAL = (*WAL)(nil)

// Size returns the current size of the WAL content.
func (w *WAL) Size() (int64, error) {
	val, err := w.storage.Size()
	if err != nil {
		return 0, err
	}
	return val - types.WALHeaderSize, nil
}

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
	return &WAL{formatter: format, storage: store, buffer: make([]types.WalLogEntry, 0, 4096)}, nil
}

func (w *WAL) LogDraw(item types.WalLogDrawItem) error {
	w.buffer = append(w.buffer, &item)
	return nil
}

func (w *WAL) LogUpdate(item types.WalLogUpdateItem) error {
	w.buffer = append(w.buffer, &item)
	return nil
}

func (w *WAL) LogSnapshot(item types.WalLogSnapshotItem) error {
	w.buffer = append(w.buffer, &item)
	return nil
}

func (w *WAL) Close() error {
	return w.storage.Close()
}

func (w *WAL) Reset() {
	w.buffer = w.buffer[:0]
}

func (w *WAL) Rotate(path string) error {
	return w.storage.Rotate(path)
}

// ParseWAL reads the WAL log file, decodes its content, and returns the log entries and the header.
func ParseWAL(path string, format types.LogFormatter) ([]types.WalLogEntry, *types.WALHeader, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil // Return empty if file doesn't exist
		}
		return nil, nil, err
	}
	defer f.Close()

	// Read header
	hdrBytes := make([]byte, types.WALHeaderSize)

	// Use io.ReadFull to ensure we read the whole header
	n, err := io.ReadFull(f, hdrBytes)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			// File is smaller than a header, so it's empty/invalid
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to read WAL header (read %d bytes): %w", n, err)
	}

	var hdr types.WALHeader
	if err := binary.Read(bytes.NewReader(hdrBytes), binary.LittleEndian, &hdr); err != nil {
		return nil, nil, fmt.Errorf("failed to decode WAL header: %w", err)
	}

	// Basic validation
	if hdr.Magic != types.WALMagic {
		return nil, nil, fmt.Errorf("invalid WAL magic number")
	}

	// Read data
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, &hdr, fmt.Errorf("failed to read WAL data: %w", err)
	}

	if len(data) == 0 {
		return []types.WalLogEntry{}, &hdr, nil
	}

	entries, err := format.Decode(data)
	if err != nil {
		return nil, &hdr, err
	}

	return entries, &hdr, nil
}
