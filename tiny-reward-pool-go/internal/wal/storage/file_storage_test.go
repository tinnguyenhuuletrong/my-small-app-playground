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
	fs, err := storage.NewFileStorage(path)
	require.NoError(t, err)
	require.NotNil(t, fs)

	// Test Write
	data := []byte("hello world")
	err = fs.Write(data)
	require.NoError(t, err)

	// Test Flush
	err = fs.Flush()
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

	// Test Rotate
	achivedPath := filepath.Join(tempDir, "test_achived.log")
	err = fs.Rotate(achivedPath)
	require.NoError(t, err)

	// Verify content of the old, rotated file
	archivedFile, err := os.Open(achivedPath)
	require.NoError(t, err)
	_, err = archivedFile.Seek(types.WALHeaderSize, io.SeekStart)
	require.NoError(t, err)
	archivedContent, err := io.ReadAll(archivedFile)
	require.NoError(t, err)
	archivedFile.Close()
	assert.Equal(t, data, archivedContent)

	// Write to the new file at the original path
	newData := []byte("hello new world")
	err = fs.Write(newData)
	require.NoError(t, err)
	err = fs.Flush()
	require.NoError(t, err)

	// Verify new file content at the original path
	newFile, err := os.Open(path)
	require.NoError(t, err)
	_, err = newFile.Seek(types.WALHeaderSize, io.SeekStart)
	require.NoError(t, err)
	newContent, err := io.ReadAll(newFile)
	require.NoError(t, err)
	newFile.Close()
	assert.Equal(t, newData, newContent)

	// Test Close
	err = fs.Close()
	require.NoError(t, err)
}
