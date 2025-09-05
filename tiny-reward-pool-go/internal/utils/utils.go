package utils

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// DefaultUtils provides a default implementation for the types.Utils interface.
// It includes a standard logger and generates paths for WALs and snapshots.

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

// GetWALFiles scans the WAL directory, finds all WAL files, and returns their paths sorted by sequence number.
func (u *DefaultUtils) GetWALFiles() ([]string, error) {
	if u.walDir == "" {
		return []string{}, nil
	}

	files, err := os.ReadDir(u.walDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL directory: %w", err)
	}

	var walFiles []string
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasPrefix(file.Name(), types.WALBaseName+".") {
			walFiles = append(walFiles, file.Name())
		}
	}

	sort.Slice(walFiles, func(i, j int) bool {
		extI := strings.TrimPrefix(filepath.Ext(walFiles[i]), ".")
		extJ := strings.TrimPrefix(filepath.Ext(walFiles[j]), ".")
		numI, _ := strconv.Atoi(extI)
		numJ, _ := strconv.Atoi(extJ)
		return numI < numJ
	})

	for i, file := range walFiles {
		walFiles[i] = filepath.Join(u.walDir, file)
	}

	return walFiles, nil
}

// GenNextWALPath determines the next available WAL sequence number and returns the corresponding path.
func (u *DefaultUtils) GenNextWALPath() (string, uint64, error) {
	walFiles, err := u.GetWALFiles()
	if err != nil {
		return "", 0, err
	}

	if len(walFiles) == 0 {
		path := filepath.Join(u.walDir, fmt.Sprintf("%s.%03d", types.WALBaseName, 0))
		return path, 0, nil
	}

	lastFile := walFiles[len(walFiles)-1]
	ext := strings.TrimPrefix(filepath.Ext(lastFile), ".")
	lastSeq, err := strconv.ParseUint(ext, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid WAL file name format: %s", lastFile)
	}

	nextSeq := lastSeq + 1
	path := filepath.Join(u.walDir, fmt.Sprintf("%s.%03d", types.WALBaseName, nextSeq))
	return path, nextSeq, nil
}


func ReadFileContent(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// MMap remaining buffer
	return bytes.TrimRight(data, "\x00"), nil
}