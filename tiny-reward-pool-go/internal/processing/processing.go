package processing

import (
	"sync"
	"sync/atomic"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type DrawRequest struct {
	RequestID uint64
	Callback  func(DrawResponse)
}

type DrawResponse struct {
	RequestID uint64
	Item      *types.PoolReward
	Err       error
}

type Processor struct {
	ctx             *types.Context
	pool            types.RewardPool
	requestID       uint64
	reqChan         chan DrawRequest
	stopChan        chan struct{}
	wg              sync.WaitGroup
	flushAfterNDraw int
	stagedDraws     int
}

type ProcessorOptional struct {
	FlushAfterNDraw int
	// Add more optional fields here in future
}

// NewProcessor creates a new Processor. Optional parameters are set via ProcessorOptional.
func NewProcessor(ctx *types.Context, pool types.RewardPool, opt *ProcessorOptional) *Processor {
	n := 10
	if opt != nil && opt.FlushAfterNDraw > 0 {
		n = opt.FlushAfterNDraw
	}
	p := &Processor{
		ctx:             ctx,
		pool:            pool,
		reqChan:         make(chan DrawRequest, 100),
		stopChan:        make(chan struct{}),
		flushAfterNDraw: n,
	}
	p.wg.Add(1)
	go p.run()
	return p
}

func (p *Processor) run() {
	defer p.wg.Done()
	for {
		select {
		case req := <-p.reqChan:
			item, err := p.pool.SelectItem(p.ctx)
			var walErr error
			success := false
			if item != nil && err == nil {
				walErr = p.ctx.WAL.LogDraw(types.WalLogItem{RequestID: req.RequestID, ItemID: item.ItemID, Success: true})
				p.stagedDraws++
			} else {
				walErr = p.ctx.WAL.LogDraw(types.WalLogItem{RequestID: req.RequestID, ItemID: "", Success: false})
				p.stagedDraws++
			}
			// Batch flush/commit after N draws
			if p.stagedDraws >= p.flushAfterNDraw {
				flushErr := p.ctx.WAL.Flush()
				if walErr == nil && flushErr == nil {
					p.pool.CommitDraw()
					success = true
				} else {
					p.pool.RevertDraw()
				}
				p.stagedDraws = 0
			}
			resp := DrawResponse{RequestID: req.RequestID, Item: nil, Err: err}
			if success {
				resp.Item = item
			} else if walErr != nil {
				resp.Err = walErr
			}
			if req.Callback != nil {
				req.Callback(resp)
			}
		case <-p.stopChan:
			// Final flush/commit on shutdown if any staged draws remain
			if p.stagedDraws > 0 {
				flushErr := p.ctx.WAL.Flush()
				if flushErr == nil {
					p.pool.CommitDraw()
				} else {
					p.pool.RevertDraw()
				}
				p.stagedDraws = 0
			}
			return
		}
	}
}

func (p *Processor) Draw(callback func(DrawResponse)) uint64 {
	reqID := atomic.AddUint64(&p.requestID, 1)
	p.reqChan <- DrawRequest{RequestID: reqID, Callback: callback}
	return reqID
}

func (p *Processor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}
