package walstream

import (
	"encoding/json"
	"log/slog"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

// LogStreamer is a WALStreamer that logs the WAL entries using the standard logger.
// This is for testing and demonstration purposes.
type LogStreamer struct {
	logger *slog.Logger
}

// NewLogStreamer creates a new LogStreamer.
func NewLogStreamer(logger *slog.Logger) *LogStreamer {
	return &LogStreamer{logger: logger}
}

// Stream logs the WAL entry.
func (s *LogStreamer) Stream(log types.WalLogEntry) {
	b, err := json.Marshal(log)
	if err != nil {
		s.logger.Error("failed to marshal log entry", "error", err)
		return
	}
	s.logger.Info("streaming wal log", "log", string(b))
}
