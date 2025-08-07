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
	ctx       *types.Context
	pool      types.RewardPool
	requestID uint64
	reqChan   chan DrawRequest
	stopChan  chan struct{}
	wg        sync.WaitGroup
}

func NewProcessor(ctx *types.Context, pool types.RewardPool) *Processor {
	p := &Processor{
		ctx:      ctx,
		pool:     pool,
		reqChan:  make(chan DrawRequest, 100),
		stopChan: make(chan struct{}),
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
				if walErr == nil {
					p.pool.CommitDraw(item.ItemID)
					success = true
				} else {
					p.pool.RevertDraw(item.ItemID)
				}
			} else {
				walErr = p.ctx.WAL.LogDraw(types.WalLogItem{RequestID: req.RequestID, ItemID: "", Success: false})
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
