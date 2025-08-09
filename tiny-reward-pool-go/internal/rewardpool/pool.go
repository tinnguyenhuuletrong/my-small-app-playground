package rewardpool

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type Pool struct {
	catalog      []types.PoolReward
	pendingDraws map[string]int
	selector     types.ItemSelector
}

var _ types.RewardPool = (*Pool)(nil)

type poolSnapshot struct {
	Catalog []types.PoolReward `json:"catalog"`
}

type PoolOptional struct {
	Selector types.ItemSelector
}

func NewPool(Catalog []types.PoolReward, ops ...PoolOptional) *Pool {
	var sel types.ItemSelector
	for _, o := range ops {
		if o.Selector != nil {
			sel = o.Selector
		}
	}

	if sel == nil {
		sel = selector.NewFenwickTreeSelector()
	}

	pool := &Pool{
		catalog:      Catalog,
		pendingDraws: make(map[string]int),
		selector:     sel,
	}
	pool.selector.Reset(Catalog)
	return pool
}

func (p *Pool) Load(config types.ConfigPool) error {
	p.catalog = config.Catalog
	p.pendingDraws = make(map[string]int)
	p.selector.Reset(p.catalog)
	return nil
}

func (p *Pool) SaveSnapshot(path string) error {
	if len(p.pendingDraws) > 0 {
		return types.ErrPendingDrawsNotEmpty
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
	p.selector.Reset(p.catalog)
	return nil
}

func (p *Pool) GetItemRemaining(ItemID string) int {
	return p.selector.GetItemRemaining(ItemID)
}

// SelectItem stages an item for draw if available
func (p *Pool) SelectItem(ctx *types.Context) (string, error) {
	selectedItemID, err := p.selector.Select(ctx)
	if err != nil {
		return "", err
	}

	p.pendingDraws[selectedItemID]++
	// Immediately decrement the quantity in the selector to prevent over-draws
	p.selector.Update(selectedItemID, -1)

	return selectedItemID, nil
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
	for itemID, count := range p.pendingDraws {
		p.selector.Update(itemID, int64(count)) // Re-add the quantity to the selector
	}
	p.pendingDraws = make(map[string]int)
}

// ApplyDrawLog decrements the quantity for a given itemID if available (internal use only)
func (p *Pool) ApplyDrawLog(itemID string) {
	for i := range p.catalog {
		if p.catalog[i].ItemID == itemID && p.catalog[i].Quantity > 0 {
			p.catalog[i].Quantity--
			p.selector.Update(itemID, -1) // Decrement in selector as well
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

	pool := NewPool(data.Catalog)

	return pool, nil
}
