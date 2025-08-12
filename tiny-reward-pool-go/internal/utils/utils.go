package utils

import (
	"bytes"
	"log/slog"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type UtilsImpl struct {
}

var _ types.Utils = (*UtilsImpl)(nil)

// GetLogger implements types.Utils.
func (u *UtilsImpl) GetLogger() *slog.Logger {
	return nil
}

func ReadFileContent(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// MMap remaining buffer
	return bytes.TrimRight(data, "\x00"), nil
}
