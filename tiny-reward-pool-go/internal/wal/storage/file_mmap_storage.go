package storage

import (
	"bytes"
	"encoding/binary"
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

func NewFileMMapStorage(path string, seqNo uint64, opts ...FileMMapStorageOps) (*FileMMapStorage, error) {
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

	currentSize := info.Size()
	isNewFile := currentSize == 0

	if isNewFile {
		if err := f.Truncate(sizeMapInBytes); err != nil {
			f.Close()
			return nil, fmt.Errorf("failed to truncate file: %w", err)
		}
	} else {
		// If the file exists, use its size for the mapping
		sizeMapInBytes = currentSize
	}

	m, err := mmap.Map(f, mmap.RDWR, 0)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("failed to mmap file: %w", err)
	}

	s := &FileMMapStorage{
		file:           f,
		mmap:           m,
		path:           path,
		sizeMapInBytes: sizeMapInBytes,
	}

	if isNewFile {
		hdr := types.WALHeader{
			Magic:   types.WALMagic,
			Version: types.WALVersion1,
			Status:  types.WALStatusOpen,
			SeqNo:   seqNo,
		}
		var buf bytes.Buffer
		if err := binary.Write(&buf, binary.LittleEndian, &hdr); err != nil {
			s.Close()
			return nil, err
		}
		copy(s.mmap, buf.Bytes())
		s.offset = int64(types.WALHeaderSize)
	} else {
		// Existing file, read header to restore offset
		var hdr types.WALHeader
		if err := binary.Read(bytes.NewReader(m[:types.WALHeaderSize]), binary.LittleEndian, &hdr); err != nil {
			s.Close()
			return nil, fmt.Errorf("failed to read WAL header from existing file: %w", err)
		}
		s.offset = int64(types.WALHeaderSize + hdr.DataLength)
	}

	return s, nil
}

func (s *FileMMapStorage) Write(data []byte) error {
	copy(s.mmap[s.offset:], data)
	s.offset += int64(len(data))
	return nil
}

func (s *FileMMapStorage) CanWrite(size int) bool {
	// For mmap, the capacity is the total length of the map.
	return s.offset+int64(size) <= int64(len(s.mmap))
}

func (s *FileMMapStorage) Size() (int64, error) {
	return s.offset, nil
}

func (s *FileMMapStorage) Flush() error {
	return s.mmap.Flush()
}

func (s *FileMMapStorage) FinalizeAndClose() error {
	if s.mmap == nil {
		return nil
	}

	if err := s.mmap.Flush(); err != nil {
		return err
	}

	hdr := types.WALHeader{
		Magic:      types.WALMagic,
		Version:    types.WALVersion1,
		Status:     types.WALStatusClosed,
		DataLength: uint64(s.offset - types.WALHeaderSize),
	}

	// Read the original SeqNo from the header before overwriting
	var originalHdr types.WALHeader
	if err := binary.Read(bytes.NewReader(s.mmap[:types.WALHeaderSize]), binary.LittleEndian, &originalHdr); err == nil {
		hdr.SeqNo = originalHdr.SeqNo
	}

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, &hdr); err != nil {
		return err
	}
	copy(s.mmap, buf.Bytes())

	if err := s.mmap.Flush(); err != nil {
		return err
	}

	if err := s.mmap.Unmap(); err != nil {
		s.file.Close()
		return err
	}

	return s.file.Close()
}

func (s *FileMMapStorage) Close() error {
	return s.FinalizeAndClose()
}
