package utils

import (
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
	return os.ReadFile(path)
}
