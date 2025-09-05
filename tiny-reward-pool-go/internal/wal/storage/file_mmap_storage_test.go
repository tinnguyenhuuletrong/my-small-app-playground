package storage_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestFileMMapStorage(t *testing.T) {
	path := "test_mmap.log"
	defer os.Remove(path)

	// Test NewFileMMapStorage
	fs, err := storage.NewFileMMapStorage(path, 0)
	assert.NoError(t, err)
	assert.NotNil(t, fs)

	// Write initial data
	initialData := []byte("initial data")
	err = fs.Write(initialData)
	assert.NoError(t, err)
	err = fs.Flush()
	assert.NoError(t, err)

	// Close the storage to ensure data is written to disk
	err = fs.Close()
	assert.NoError(t, err)

	// Verify original file content
	originalContent, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(originalContent), string(initialData))
}