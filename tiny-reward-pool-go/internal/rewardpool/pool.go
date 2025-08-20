package rewardpool

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/selector"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type Pool struct {
	pendingDraws map[string]int
	selector     types.ItemSelector
}

var _ types.RewardPool = (*Pool)(nil)

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
		pendingDraws: make(map[string]int),
		selector:     sel,
	}

	copyCatalog := Catalog
	pool.selector.Reset(copyCatalog)
	return pool
}

func (p *Pool) Load(config types.ConfigPool) error {
	p.pendingDraws = make(map[string]int)
	p.selector.Reset(config.Catalog)
	return nil
}

func (p *Pool) CreateSnapshot() (*types.PoolSnapshot, error) {
	if len(p.pendingDraws) > 0 {
		return nil, types.ErrPendingDrawsNotEmpty
	}

	// Reflect item remaining
	snapshot_catalog := p.selector.SnapshotCatalog()

	snap := &types.PoolSnapshot{
		Catalog: snapshot_catalog,
	}
	return snap, nil
}

func (p *Pool) LoadSnapshot(snapshot *types.PoolSnapshot) error {
	p.pendingDraws = make(map[string]int)
	p.selector.Reset(snapshot.Catalog)
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
	// p.pendingDraws = make(map[string]int)
	clear(p.pendingDraws)
}

// RevertDraw cancels a staged draw
func (p *Pool) RevertDraw() {
	for itemID, count := range p.pendingDraws {
		p.selector.Update(itemID, int64(count)) // Re-add the quantity to the selector
	}
	// p.pendingDraws = make(map[string]int)
	clear(p.pendingDraws)
}

// ApplyDrawLog decrements the quantity for a given itemID if available (internal use only)
func (p *Pool) ApplyDrawLog(itemID string) {
	if p.selector.GetItemRemaining((itemID)) > 0 {
		p.selector.Update(itemID, -1) // Decrement in selector as well
	}
}

func (p *Pool) ApplyUpdateLog(itemID string, quantity int, probability int64) {
	p.selector.UpdateItem(itemID, quantity, probability)
}

func (p *Pool) UpdateItem(itemID string, quantity int, probability int64) error {
	p.selector.UpdateItem(itemID, quantity, probability)
	return nil
}

func (p *Pool) State() []types.PoolReward {
	catalog := p.selector.SnapshotCatalog()
	return catalog
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
