package storage

import (
	"math"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type FileStorage struct {
	file     *os.File
	capacity int
	usage    int
}

var _ types.Storage = (*FileStorage)(nil)

type FileStorageOpt struct {
	SizeFileInBytes int
}

func NewFileStorage(path string, ops ...FileStorageOpt) (*FileStorage, error) {
	maxEntry := math.MaxInt
	for _, v := range ops {
		maxEntry = v.SizeFileInBytes
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileStorage{file: f, capacity: maxEntry}, nil
}

func (s *FileStorage) Write(data []byte) error {
	if _, err := s.file.Write(data); err != nil {
		return err
	}
	s.usage += len(data)
	return nil
}

func (s *FileStorage) CanWrite(size int) bool {
	return s.usage+size <= s.capacity
}

func (s *FileStorage) Size() (int64, error) {
	return int64(s.usage), nil
}

func (s *FileStorage) Flush() error {
	return s.file.Sync()
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Rotate(archivePath string) error {
	// Get the path of the current file.
	originalPath := s.file.Name()

	// Close the current file.
	if err := s.file.Close(); err != nil {
		return err
	}

	// Rename the old file to the new path (archive it).
	if err := os.Rename(originalPath, archivePath); err != nil {
		return err
	}

	// Create a new file at the original path.
	newFile, err := os.OpenFile(originalPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	// Update the storage with the new file.
	s.file = newFile
	s.usage = 0
	return nil
}
