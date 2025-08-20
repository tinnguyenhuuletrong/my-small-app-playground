package walstream

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

func TestLogStreamer_Stream(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	streamer := NewLogStreamer(logger)

	logEntry := &types.WalLogDrawItem{
		WalLogEntryBase: types.WalLogEntryBase{Type: types.LogTypeDraw},
		RequestID:       1,
		ItemID:          "gold",
		Success:         true,
	}

	streamer.Stream(logEntry)

	var logOutput map[string]interface{}
	err := json.Unmarshal(buf.Bytes(), &logOutput)
	require.NoError(t, err)

	assert.Equal(t, "streaming wal log", logOutput["msg"])

	logField, ok := logOutput["log"].(string)
	require.True(t, ok)

	var innerLog map[string]interface{}
	err = json.Unmarshal([]byte(logField), &innerLog)
	require.NoError(t, err)

	assert.Equal(t, float64(1), innerLog["request_id"])
	assert.Equal(t, "gold", innerLog["item_id"])
}