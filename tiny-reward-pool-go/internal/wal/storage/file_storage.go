package storage

import (
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type FileStorage struct {
	file *os.File
}

var _ types.Storage = (*FileStorage)(nil)

func NewFileStorage(path string) (*FileStorage, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileStorage{file: f}, nil
}

func (s *FileStorage) Write(data []byte) error {
	if _, err := s.file.Write(data); err != nil {
		return err
	}
	return nil
}

func (s *FileStorage) Flush() error {
	return s.file.Sync()
}

func (s *FileStorage) Close() error {
	return s.file.Close()
}

func (s *FileStorage) Rotate(newPath string) error {
	f, err := os.OpenFile(newPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	s.file.Close()
	s.file = f
	return nil
}
