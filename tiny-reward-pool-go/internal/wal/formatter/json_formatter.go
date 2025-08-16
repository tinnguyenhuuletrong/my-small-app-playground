package formatter

import (
	"encoding/json"
	"fmt"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type JSONFormatter struct{}

var _ types.LogFormatter = (*JSONFormatter)(nil)

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Encode(items []types.WalLogEntry) ([]byte, error) {
	var encodedData []byte
	for _, item := range items {
		data, err := json.Marshal(item)
		if err != nil {
			return nil, err
		}
		encodedData = append(encodedData, data...)
		encodedData = append(encodedData, '\n') // Add newline for JSONL format
	}
	return encodedData, nil
}

// walLogEntryWrapper is a helper struct to unmarshal polymorphic WalLogEntry types.
type walLogEntryWrapper struct {
	types.WalLogEntry
}

func (w *walLogEntryWrapper) UnmarshalJSON(data []byte) error {
	type typeFinder struct {
		Type types.LogType `json:"type"`
	}
	var tf typeFinder
	if err := json.Unmarshal(data, &tf); err != nil {
		return fmt.Errorf("failed to find type: %w", err)
	}

	var entry types.WalLogEntry
	switch tf.Type {
	case types.LogTypeDraw:
		entry = &types.WalLogDrawItem{}
	case types.LogTypeUpdate:
		entry = &types.WalLogUpdateItem{}
	case types.LogTypeSnapshot:
		entry = &types.WalLogSnapshotItem{}
	case types.LogTypeRotate:
		entry = &types.WalLogRotateItem{}
	default:
		return fmt.Errorf("unknown log type: %d", tf.Type)
	}

	if err := json.Unmarshal(data, entry); err != nil {
		return err
	}
	w.WalLogEntry = entry
	return nil
}

func (f *JSONFormatter) Decode(data []byte) ([]types.WalLogEntry, error) {
	var items []types.WalLogEntry
	lines := splitLines(data)

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		var wrapper walLogEntryWrapper
		if err := json.Unmarshal(line, &wrapper); err != nil {
			return nil, err
		}

		items = append(items, wrapper.WalLogEntry)
	}
	return items, nil
}

// splitLines splits a byte slice into lines, handling both \n and \r\n
func splitLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			lines = append(lines, data[start:i])
			start = i + 1
		} else if b == '\r' && i+1 < len(data) && data[i+1] == '\n' {
			lines = append(lines, data[start:i])
			start = i + 2
			i++ // Skip the \n
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}