package rewardpool

import (
	"encoding/json"
	"os"
)

// Item represents a reward item in the pool
type Item struct {
	ItemID      string  `json:"item_id"`
	Quantity    int     `json:"quantity"`
	Probability float64 `json:"probability"`
}

// Pool holds all items
type Pool struct {
	Catalog []Item
}

// LoadPool loads pool from config.json
func LoadPool(configPath string) (*Pool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data struct {
		Catalog []Item `json:"catalog"`
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return &Pool{Catalog: data.Catalog}, nil
}
