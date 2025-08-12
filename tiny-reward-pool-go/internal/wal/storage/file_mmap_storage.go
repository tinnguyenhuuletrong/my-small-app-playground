package storage

import (
	"fmt"
	"os"

	"github.com/edsrzf/mmap-go"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

const ( // Constants for mmap file operations
	defaultMmapFileSize = 1024 * 1024 * 10 // 10 MB
)

type FileMMapStorage struct {
	file   *os.File
	mmap   mmap.MMap
	path   string
	offset int64
}

var _ types.Storage = (*FileMMapStorage)(nil)

func NewFileMMapStorage(path string) (*FileMMapStorage, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	offset := info.Size()

	if offset == 0 {
		if err := f.Truncate(defaultMmapFileSize); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to truncate file: %w", err)
		}
		offset = 0
	}

	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	return &FileMMapStorage{
		file:   f,
		mmap:   m,
		path:   path,
		offset: offset,
	}, nil
}

func (s *FileMMapStorage) Write(data []byte) error {
	// Ensure enough space in mmap
	if s.offset+int64(len(data)) > int64(len(s.mmap)) {
		// Re-mmap with larger size or handle error
		return fmt.Errorf("mmap buffer full, cannot write %d bytes", len(data))
	}
	copy(s.mmap[s.offset:], data)
	s.offset += int64(len(data))
	return nil
}

func (s *FileMMapStorage) Flush() error {
	return s.mmap.Flush()
}

func (s *FileMMapStorage) Close() error {
	if s.mmap != nil {
		if err := s.mmap.Unmap(); err != nil {
			return err
		}
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *FileMMapStorage) Rotate(newPath string) error {
	// Unmap and close current file
	if err := s.Close(); err != nil {
		return err
	}

	// Rename old file (optional, depending on desired rotation behavior)
	// For simplicity, we'll just open a new file at newPath

	// Open new file and mmap it
	f, err := os.OpenFile(newPath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	// Truncate new file if it's empty
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	offset := info.Size()
	if offset == 0 {
		if err := f.Truncate(defaultMmapFileSize); err != nil {
			f.Close()
			return fmt.Errorf("failed to truncate new file: %w", err)
		}
	}

	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		f.Close()
		return fmt.Errorf("failed to mmap new file: %w", err)
	}

	s.file = f
	s.mmap = m
	s.path = newPath
	s.offset = 0 // Reset offset for the new file

	return nil
}
