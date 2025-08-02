package rewardpool

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type Pool struct {
	Catalog []types.PoolReward
}

func (p *Pool) Load(config types.ConfigPool) error {
	p.Catalog = config.Catalog
	return nil
}

func (p *Pool) Draw(ctx *types.Context) (*types.PoolReward, error) {
	idx, err := ctx.Utils.RandomItem(p.Catalog)
	if err != nil {
		return nil, err
	}
	if p.Catalog[idx].Quantity <= 0 {
		return nil, nil
	}
	p.Catalog[idx].Quantity--
	logItem := types.WalLogItem{
		RequestID: 0, // should be generated
		ItemID:    p.Catalog[idx].ItemID,
		Success:   true,
	}
	ctx.WAL.LogDraw(logItem)
	return &p.Catalog[idx], nil
}

func LoadPool(configPath string) (*Pool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data types.ConfigPool
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return nil, err
	}
	return &Pool{Catalog: data.Catalog}, nil
}
