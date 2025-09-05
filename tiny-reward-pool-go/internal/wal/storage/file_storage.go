package storage

import (
	"encoding/binary"
	"io"
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

func NewFileStorage(path string, seqNo uint64, ops ...FileStorageOpt) (*FileStorage, error) {
	maxSize := math.MaxInt
	for _, v := range ops {
		if v.SizeFileInBytes > 0 {
			maxSize = v.SizeFileInBytes
		}
	}

	// Use O_RDWR instead of O_APPEND and O_WRONLY to allow seeking back to write the header
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	s := &FileStorage{file: f, capacity: maxSize}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if info.Size() == 0 {
		// New file, write header
		hdr := types.WALHeader{
			Magic:   types.WALMagic,
			Version: types.WALVersion1,
			Status:  types.WALStatusOpen,
			SeqNo:   seqNo,
		}
		if err := binary.Write(f, binary.LittleEndian, &hdr); err != nil {
			f.Close()
			return nil, err
		}
		s.usage = types.WALHeaderSize
	} else {
		// Existing file, just record usage
		s.usage = int(info.Size())
	}

	// Seek to the end for subsequent writes
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		f.Close()
		return nil, err
	}

	return s, nil
}

func (s *FileStorage) Write(data []byte) error {
	n, err := s.file.Write(data)
	if err != nil {
		return err
	}
	s.usage += n
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

func (s *FileStorage) FinalizeAndClose() error {
	if err := s.file.Sync(); err != nil {
		return err
	}

	// Seek to the beginning to write the header
	if _, err := s.file.Seek(0, io.SeekStart); err != nil {
		return err
	}

	hdr := types.WALHeader{
		Magic:   types.WALMagic,
		Version: types.WALVersion1,
		Status:  types.WALStatusClosed,
	}

	if err := binary.Write(s.file, binary.LittleEndian, &hdr); err != nil {
		return err
	}

	if err := s.file.Sync(); err != nil {
        return err
    }

	return s.file.Close()
}

func (s *FileStorage) Close() error {
	return s.FinalizeAndClose()
}