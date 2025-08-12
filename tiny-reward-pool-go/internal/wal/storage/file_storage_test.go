package storage_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestFileStorage(t *testing.T) {
	path := "test.log"
	defer os.Remove(path)

	// Test NewFileStorage
	fs, err := storage.NewFileStorage(path)
	assert.NoError(t, err)
	assert.NotNil(t, fs)

	// Test Write
	data := []byte("hello world")
	err = fs.Write(data)
	assert.NoError(t, err)

	// Test Flush
	err = fs.Flush()
	assert.NoError(t, err)

	// Verify content
	content, err := os.ReadFile(path)
	assert.NoError(t, err)
	assert.Equal(t, data, content)

	// Test Rotate
	newPath := "test_new.log"
	defer os.Remove(newPath)
	err = fs.Rotate(newPath)
	assert.NoError(t, err)

	// Write to new file
	newData := []byte("hello new world")
	err = fs.Write(newData)
	assert.NoError(t, err)
	err = fs.Flush()
	assert.NoError(t, err)

	// Verify new file content
	newContent, err := os.ReadFile(newPath)
	assert.NoError(t, err)
	assert.Equal(t, newData, newContent)

	// Test Close
	err = fs.Close()
	assert.NoError(t, err)
}
