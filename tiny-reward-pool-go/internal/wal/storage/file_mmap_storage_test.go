package storage_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestFileMMapStorage(t *testing.T) {
	path := "test_mmap.log"
	achivedPath := "test_mmap_achived.log"
	defer os.Remove(path)
	defer os.Remove(achivedPath)

	// Test NewFileMMapStorage
	fs, err := storage.NewFileMMapStorage(path)
	assert.NoError(t, err)
	assert.NotNil(t, fs)

	// Write initial data
	initialData := []byte("initial data")
	err = fs.Write(initialData)
	assert.NoError(t, err)
	err = fs.Flush()
	assert.NoError(t, err)

	// Close the storage to ensure data is written to disk
	err = fs.Flush()
	assert.NoError(t, err)

	// Verify original file content
	originalContent, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(originalContent), string(initialData))

	// Test Rotate
	err = fs.Rotate(achivedPath)
	assert.NoError(t, err)

	// Verify the content of the archived file
	archivedContent, err := os.ReadFile(achivedPath)
	assert.NoError(t, err)
	assert.Contains(t, string(archivedContent), string(initialData))

	// Write new data after rotation to the original path
	newData := []byte("new data")
	err = fs.Write(newData)
	assert.NoError(t, err)
	err = fs.Flush()
	assert.NoError(t, err)

	// Verify the content of the new file at the original path
	newContent, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Contains(t, string(newContent), string(newData))

	// Close the storage
	err = fs.Close()
}
