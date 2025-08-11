package storage

import (
	"bufio"
	"os"
)

type FileStorage struct {
	file *os.File
}

func NewFileStorage(path string) (*FileStorage, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &FileStorage{file: f}, nil
}

func (s *FileStorage) WriteAll(data [][]byte) error {
	for _, d := range data {
		if _, err := s.file.Write(d); err != nil {
			return err
		}
	}
	return nil
}

func (s *FileStorage) ReadAll() ([][]byte, error) {
	// Reopen file for reading from the beginning
	currentPath := s.file.Name()
	s.file.Close()

	f, err := os.Open(currentPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // Return empty slice if WAL file doesn't exist
		}
		return nil, err
	}
	defer f.Close()

	var lines [][]byte
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Bytes())
	}
	return lines, scanner.Err()
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
