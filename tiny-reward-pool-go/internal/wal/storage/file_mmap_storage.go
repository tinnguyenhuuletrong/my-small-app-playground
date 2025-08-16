package storage

import (
	"fmt"
	"os"

	"github.com/edsrzf/mmap-go"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

const ( // Constants for mmap file operations
	defaultMmapFileSize int64 = 1024 * 1024 * 10 // 10 MB
)

type FileMMapStorage struct {
	file   *os.File
	mmap   mmap.MMap
	path   string
	offset int64

	sizeMapInBytes int64
}

var _ types.Storage = (*FileMMapStorage)(nil)

type FileMMapStorageOps struct {
	MMapFileSizeInBytes int64
}

func NewFileMMapStorage(path string, opts ...FileMMapStorageOps) (*FileMMapStorage, error) {
	sizeMapInBytes := defaultMmapFileSize
	for _, val := range opts {
		if val.MMapFileSizeInBytes > 0 {
			sizeMapInBytes = val.MMapFileSizeInBytes
		}
	}

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
		if err := f.Truncate(sizeMapInBytes); err != nil {
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
		file:           f,
		mmap:           m,
		path:           path,
		offset:         offset,
		sizeMapInBytes: sizeMapInBytes,
	}, nil
}

func (s *FileMMapStorage) Write(data []byte) error {
	copy(s.mmap[s.offset:], data)
	s.offset += int64(len(data))
	return nil
}

func (s *FileMMapStorage) CanWrite(size int) bool {
	return s.offset+int64(size) <= int64(len(s.mmap))
}

func (s *FileMMapStorage) Size() (int64, error) {
	return s.offset, nil
}

func (s *FileMMapStorage) Flush() error {
	return s.mmap.Flush()
}

func (s *FileMMapStorage) Close() error {
	if s.mmap != nil {
		if err := s.mmap.Unmap(); err != nil {
			return err
		}
		s.mmap = nil
	}
	if s.file != nil {
		return s.file.Close()
	}
	return nil
}

func (s *FileMMapStorage) Rotate(archivePath string) error {
	// Unmap and close current file
	if err := s.Close(); err != nil {
		return err
	}

	// Rename old file to the new archive path
	if err := os.Rename(s.path, archivePath); err != nil {
		return err
	}

	// Re-initialize the storage at the original path
	newStorage, err := NewFileMMapStorage(s.path, FileMMapStorageOps{MMapFileSizeInBytes: s.sizeMapInBytes})
	if err != nil {
		return fmt.Errorf("failed to re-initialize mmap storage after rotate: %w", err)
	}

	// Update the current storage instance with the new one
	*s = *newStorage

	return nil
}
