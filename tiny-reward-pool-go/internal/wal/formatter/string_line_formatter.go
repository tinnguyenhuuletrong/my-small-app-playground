package formatter

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type StringLineFormatter struct{}

var _ types.LogFormatter = (*StringLineFormatter)(nil)

func NewStringLineFormatter() *StringLineFormatter {
	return &StringLineFormatter{}
}

func (f *StringLineFormatter) Encode(items []types.WalLogEntry) ([]byte, error) {
	var sb strings.Builder
	for _, item := range items {
		switch v := item.(type) {
		case *types.WalLogDrawItem:
			sb.WriteString(fmt.Sprintf("%d,%d,%s,%d,%t\n", item.GetType(), v.RequestID, v.ItemID, v.Error, v.Success))
		case *types.WalLogUpdateItem:
			sb.WriteString(fmt.Sprintf("%d,%s,%d,%d\n", item.GetType(), v.ItemID, v.Quantity, v.Probability))
		case *types.WalLogSnapshotItem:
			sb.WriteString(fmt.Sprintf("%d,%s\n", item.GetType(), v.Path))
		case *types.WalLogRotateItem:
			sb.WriteString(fmt.Sprintf("%d,%s,%s\n", item.GetType(), v.OldPath, v.NewPath))
		}
	}
	return []byte(sb.String()), nil
}

func (f *StringLineFormatter) Decode(data []byte) ([]types.WalLogEntry, error) {
	var items []types.WalLogEntry
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(line, ",")
		if len(parts) < 1 {
			return nil, fmt.Errorf("invalid WAL log format: %s", line)
		}

		typeVal, err := strconv.Atoi(parts[0])
		if err != nil {
			return nil, fmt.Errorf("invalid type in WAL log: %s", parts[0])
		}

		logType := types.LogType(typeVal)

		switch logType {
		case types.LogTypeDraw:
			if len(parts) != 5 {
				return nil, fmt.Errorf("invalid WAL log format for draw: %s", line)
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
			items = append(items, &types.WalLogDrawItem{
				WalLogEntryBase: types.WalLogEntryBase{
					Type:  logType,
					Error: types.LogError(errorVal),
				},
				RequestID: requestID,
				ItemID:    itemID,
				Success:   success,
			})
		case types.LogTypeUpdate:
			if len(parts) != 4 {
				return nil, fmt.Errorf("invalid WAL log format for update: %s", line)
			}
			itemID := parts[1]
			quantity, err := strconv.Atoi(parts[2])
			if err != nil {
				return nil, fmt.Errorf("invalid quantity in WAL log: %s", parts[2])
			}
			probability, err := strconv.ParseInt(parts[3], 10, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid probability in WAL log: %s", parts[3])
			}
			items = append(items, &types.WalLogUpdateItem{
				WalLogEntryBase: types.WalLogEntryBase{
					Type: logType,
				},
				ItemID:      itemID,
				Quantity:    quantity,
				Probability: probability,
			})
		case types.LogTypeSnapshot:
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid WAL log format for snapshot: %s", line)
			}
			items = append(items, &types.WalLogSnapshotItem{
				WalLogEntryBase: types.WalLogEntryBase{
					Type: logType,
				},
				Path: parts[1],
			})
		case types.LogTypeRotate:
			if len(parts) != 3 {
				return nil, fmt.Errorf("invalid WAL log format for rotate: %s", line)
			}
			items = append(items, &types.WalLogRotateItem{
				WalLogEntryBase: types.WalLogEntryBase{
					Type: logType,
				},
				OldPath: parts[1],
				NewPath: parts[2],
			})
		}
	}
	return items, nil
}