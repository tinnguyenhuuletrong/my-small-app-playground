package storage_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
)

func TestFileMMapStorage(t *testing.T) {
	path := "test_mmap.log"
	defer os.Remove(path)

	// Test NewFileMMapStorage
	fs, err := storage.NewFileMMapStorage(path, 0)
	require.NoError(t, err)
	require.NotNil(t, fs)

	// Write initial data
	initialData := []byte("initial data")
	err = fs.Write(initialData)
	require.NoError(t, err)
	err = fs.Flush()
	require.NoError(t, err)

	// Close the storage to ensure data is written to disk
	err = fs.Close()
	require.NoError(t, err)

	// Verify original file content
	originalContent, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(originalContent), string(initialData))
}

func TestFileMMapStorage_Reopen(t *testing.T) {
	path := "test_mmap_reopen.log"
	defer os.Remove(path)

	// 1. Create and write to the mmap storage
	fs1, err := storage.NewFileMMapStorage(path, 0, storage.FileMMapStorageOps{MMapFileSizeInBytes: 1024})
	require.NoError(t, err)
	initialData := []byte("initial data")
	err = fs1.Write(initialData)
	require.NoError(t, err)
	err = fs1.Close()
	require.NoError(t, err)

	// 2. Re-open the storage
	fs2, err := storage.NewFileMMapStorage(path, 0, storage.FileMMapStorageOps{MMapFileSizeInBytes: 1024})
	require.NoError(t, err)

	// 3. Write more data
	secondData := []byte(" and more data")
	err = fs2.Write(secondData)
	require.NoError(t, err)
	err = fs2.Close()
	require.NoError(t, err)

	// 4. Verify the content
	finalContent, err := os.ReadFile(path)
	require.NoError(t, err)

	expectedContent := append(initialData, secondData...)
	assert.Contains(t, string(finalContent[types.WALHeaderSize:]), string(expectedContent))
}
