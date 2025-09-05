package storage_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestFileStorage(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "test.log")

	// Test NewFileStorage
	fs, err := storage.NewFileStorage(path, 0)
	require.NoError(t, err)
	require.NotNil(t, fs)

	// Test Write
	data := []byte("hello world")
	err = fs.Write(data)
	require.NoError(t, err)

	// Test Flush
	err = fs.Flush()
	require.NoError(t, err)

	// Test Close
	err = fs.Close()
	require.NoError(t, err)

	// Verify content
	file, err := os.Open(path)
	require.NoError(t, err)
	_, err = file.Seek(types.WALHeaderSize, io.SeekStart)
	require.NoError(t, err)
	content, err := io.ReadAll(file)
	require.NoError(t, err)
	file.Close()
	assert.Equal(t, data, content)
}