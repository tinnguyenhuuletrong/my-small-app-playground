package formatter

import (
	"encoding/json"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type JSONFormatter struct{}

var _ types.LogFormatter = (*JSONFormatter)(nil)

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Encode(items []types.WalLogDrawItem) ([]byte, error) {
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

func (f *JSONFormatter) Decode(data []byte) ([]types.WalLogDrawItem, error) {
	var items []types.WalLogDrawItem
	lines := splitLines(data)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var item types.WalLogDrawItem
		if err := json.Unmarshal(line, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
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
