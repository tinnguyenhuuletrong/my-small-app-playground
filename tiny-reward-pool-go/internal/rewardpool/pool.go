package rewardpool

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type Pool struct {
	catalog      []types.PoolReward
	pendingDraws map[string]int
}

var _ types.RewardPool = (*Pool)(nil)

type poolSnapshot struct {
	Catalog []types.PoolReward `json:"catalog"`
}

func NewPool(Catalog []types.PoolReward) *Pool {
	return &Pool{
		catalog:      Catalog,
		pendingDraws: make(map[string]int),
	}
}

func (p *Pool) Load(config types.ConfigPool) error {
	p.catalog = config.Catalog
	p.pendingDraws = make(map[string]int)
	return nil
}

func (p *Pool) SaveSnapshot(path string) error {
	if len(p.pendingDraws) > 0 {
		return fmt.Errorf("PendingDraws remaining. Please CommitDraw or RevertDraw before")
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	snap := poolSnapshot{
		Catalog: p.catalog,
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
	p.catalog = snap.Catalog
	p.pendingDraws = make(map[string]int)
	return nil
}

func (p *Pool) GetItemRemaining(ItemID string) int {
	for _, itm := range p.catalog {
		if itm.ItemID == ItemID {
			Quantity := itm.Quantity
			if stagValue, ok := p.pendingDraws[ItemID]; ok {
				return Quantity - stagValue
			}
			return Quantity
		}
	}
	return -1
}

// SelectItem stages an item for draw if available
func (p *Pool) SelectItem(ctx *types.Context) (string, error) {
	// Build a temporary catalog of available items
	var available []types.PoolReward
	for _, item := range p.catalog {
		staged := p.pendingDraws[item.ItemID]
		if item.Quantity-staged > 0 {
			available = append(available, item)
		}
	}
	if len(available) == 0 {
		return "", nil
	}
	idx, err := ctx.Utils.RandomItem(available)
	if err != nil {
		return "", err
	}
	selected := available[idx]
	p.pendingDraws[selected.ItemID]++
	// Return a copy

	return selected.ItemID, nil
}

// CommitDraw finalizes a staged draw
func (p *Pool) CommitDraw() {
	for itemID, count := range p.pendingDraws {
		for i := range p.catalog {
			if p.catalog[i].ItemID == itemID {
				if p.catalog[i].Quantity >= count {
					p.catalog[i].Quantity -= count
				} else {
					p.catalog[i].Quantity = 0
				}
				break
			}
		}
	}
	p.pendingDraws = make(map[string]int)
}

// RevertDraw cancels a staged draw
func (p *Pool) RevertDraw() {
	p.pendingDraws = make(map[string]int)
}

// applyDrawLog decrements the quantity for a given itemID if available (internal use only)
func (p *Pool) ApplyDrawLog(itemID string) {
	for i := range p.catalog {
		if p.catalog[i].ItemID == itemID && p.catalog[i].Quantity > 0 {
			p.catalog[i].Quantity--
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
