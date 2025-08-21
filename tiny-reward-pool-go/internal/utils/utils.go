package utils

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// DefaultUtils provides a default implementation for the types.Utils interface.
// It includes a standard logger and generates timestamp-based paths for WALs and snapshots.

type DefaultUtils struct {
	logger      *slog.Logger
	walDir      string
	snapshotDir string
}

var _ types.Utils = (*DefaultUtils)(nil)

// NewDefaultUtils creates a new DefaultUtils.
// It takes the base directories for WAL and snapshot files as arguments.
func NewDefaultUtils(walDir, snapshotDir string, logLevel slog.Level, writer io.Writer) *DefaultUtils {
	if writer == nil {
		writer = os.Stdout
	}
	return &DefaultUtils{
		logger:      slog.New(slog.NewTextHandler(writer, &slog.HandlerOptions{Level: logLevel})),
		walDir:      walDir,
		snapshotDir: snapshotDir,
	}
}

// GetLogger returns the logger instance.
func (u *DefaultUtils) GetLogger() *slog.Logger {
	return u.logger
}

// GenRotatedWALPath generates a new path for an archived WAL file.
// The path is timestamped, e.g., "wal-20230101T150405.log".
// It returns a pointer to the path, or nil if path generation is disabled.
func (u *DefaultUtils) GenRotatedWALPath() *string {
	if u.walDir == "" {
		return nil
	}
	timestamp := time.Now().Format("20060102T150405")
	path := filepath.Join(u.walDir, fmt.Sprintf("wal-%s.log", timestamp))
	return &path
}

// GenSnapshotPath generates a new path for a snapshot file.
// The path is fixed "snapshot.json".
// It returns a pointer to the path, or nil if path generation is disabled.
func (u *DefaultUtils) GenSnapshotPath() *string {
	if u.snapshotDir == "" {
		return nil
	}
	path := filepath.Join(u.snapshotDir, "snapshot.json")
	return &path
}

func ReadFileContent(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// MMap remaining buffer
	return bytes.TrimRight(data, "\x00"), nil
}
