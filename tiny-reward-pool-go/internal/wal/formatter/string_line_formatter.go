package formatter

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type StringLineFormatter struct{}

func NewStringLineFormatter() *StringLineFormatter {
	return &StringLineFormatter{}
}

func (f *StringLineFormatter) Encode(items []types.WalLogDrawItem) ([]byte, error) {
	var sb strings.Builder
	for _, item := range items {
		sb.WriteString(fmt.Sprintf("%d,%d,%s,%d,%t\n", item.Type, item.RequestID, item.ItemID, item.Error, item.Success))
	}
	return []byte(sb.String()), nil
}

func (f *StringLineFormatter) Decode(data []byte) ([]types.WalLogDrawItem, error) {
	var items []types.WalLogDrawItem
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) != 5 {
			return nil, fmt.Errorf("invalid WAL log format: %s", line)
		}

		typeVal, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid type in WAL log: %s", parts[0])
		}
		requestID, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid request ID in WAL log: %s", parts[1])
		}
		itemID := parts[2]
		errorVal, err := strconv.Atoi(parts[3])
		if err != nil {
			return nil, fmt.Errorf("invalid error in WAL log: %s", parts[3])
		}
		success, err := strconv.ParseBool(parts[4])
		if err != nil {
			return nil, fmt.Errorf("invalid success in WAL log: %s", parts[4])
		}

		items = append(items, types.WalLogDrawItem{
			WalLogItem: types.WalLogItem{
				Type:  types.LogType(typeVal),
				Error: types.LogError(errorVal),
			},
			RequestID: requestID,
			ItemID:    itemID,
			Success:   success,
		})
	}
	return items, nil
}