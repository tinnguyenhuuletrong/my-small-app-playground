package rewardpool

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type Pool struct {
	Catalog      []types.PoolReward
	PendingDraws map[string]int
}

type poolSnapshot struct {
	Catalog      []types.PoolReward `json:"catalog"`
	PendingDraws map[string]int     `json:"pending_draws"`
}

func (p *Pool) Load(config types.ConfigPool) error {
	p.Catalog = config.Catalog
	p.PendingDraws = make(map[string]int)
	return nil
}

func (p *Pool) SaveSnapshot(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	snap := poolSnapshot{
		Catalog:      p.Catalog,
		PendingDraws: p.PendingDraws,
	}
	enc := json.NewEncoder(file)
	return enc.Encode(snap)
}

func (p *Pool) LoadSnapshot(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	var snap poolSnapshot
	dec := json.NewDecoder(file)
	if err := dec.Decode(&snap); err != nil {
		return err
	}
	p.Catalog = snap.Catalog
	p.PendingDraws = snap.PendingDraws
	return nil
}

// SelectItem stages an item for draw if available
func (p *Pool) SelectItem(ctx *types.Context) (*types.PoolReward, error) {
	// Build a temporary catalog of available items
	var available []types.PoolReward
	for _, item := range p.Catalog {
		staged := p.PendingDraws[item.ItemID]
		if item.Quantity-staged > 0 {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		return nil, nil
	}
	idx, err := ctx.Utils.RandomItem(available)
	if err != nil {
		return nil, err
	}
	selected := available[idx]
	p.PendingDraws[selected.ItemID]++
	// Return a copy
	copyItem := selected
	return &copyItem, nil
}

// CommitDraw finalizes a staged draw
func (p *Pool) CommitDraw(itemID string) {
	for i := range p.Catalog {
		if p.Catalog[i].ItemID == itemID && p.PendingDraws[itemID] > 0 {
			if p.Catalog[i].Quantity > 0 {
				p.Catalog[i].Quantity--
			}
			p.PendingDraws[itemID]--
			break
		}
	}
}

// RevertDraw cancels a staged draw
func (p *Pool) RevertDraw(itemID string) {
	if p.PendingDraws[itemID] > 0 {
		p.PendingDraws[itemID]--
	}
}

// applyDrawLog decrements the quantity for a given itemID if available (internal use only)
func (p *Pool) ApplyDrawLog(itemID string) {
	for i := range p.Catalog {
		if p.Catalog[i].ItemID == itemID && p.Catalog[i].Quantity > 0 {
			p.Catalog[i].Quantity--
			break
		}
	}
}

func CreatePoolFromConfigPath(configPath string) (*Pool, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var data types.ConfigPool

	err = json.NewDecoder(file).Decode(&data)
	if err != nil {
		return nil, err
	}

	pool := Pool{}
	pool.Load(data)

	return &pool, nil
}
