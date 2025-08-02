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

func (p *Pool) SaveSnapshot(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := json.NewEncoder(file)
	return enc.Encode(p.Catalog)
}

func (p *Pool) LoadSnapshot(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var catalog []types.PoolReward
	dec := json.NewDecoder(file)
	if err := dec.Decode(&catalog); err != nil {
		return err
	}
	p.Catalog = catalog
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
