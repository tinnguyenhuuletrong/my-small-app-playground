package processing

import (
	"fmt"
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
	Item      string
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
	// Number of draw operations after which the processor should flush its state.
	FlushAfterNDraw int

	// Size of the buffer for incoming requests.
	RequestBufferSize int

	// Add more optional fields here in future
}

// NewProcessor creates a new Processor. Optional parameters are set via ProcessorOptional.
func NewProcessor(ctx *types.Context, pool types.RewardPool, opt *ProcessorOptional) *Processor {
	n := 10
	if opt != nil && opt.FlushAfterNDraw > 0 {
		n = opt.FlushAfterNDraw
	}
	bufSize := 100
	if opt != nil && opt.RequestBufferSize > 0 {
		bufSize = opt.RequestBufferSize
	}

	p := &Processor{
		ctx:             ctx,
		pool:            pool,
		reqChan:         make(chan DrawRequest, bufSize),
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
			if err == nil {
				walErr = p.ctx.WAL.LogDraw(types.WalLogItem{RequestID: req.RequestID, ItemID: item, Success: true})
			} else {
				walErr = p.ctx.WAL.LogDraw(types.WalLogItem{RequestID: req.RequestID, ItemID: "", Success: false})
			}
			p.stagedDraws++

			// Batch flush/commit after N draws
			if p.stagedDraws >= p.flushAfterNDraw {
				p.Flush()
			}

			// Construct the response
			resp := DrawResponse{RequestID: req.RequestID, Item: "", Err: err}
			if walErr == nil {
				resp.Item = item
			} else {
				resp.Err = walErr
			}

			if req.Callback != nil {
				req.Callback(resp)
			}
		case <-p.stopChan:
			// Graceful shutdown: stop receiving requests, cancel pending, then flush
			// Drain reqChan and cancel all pending requests
			if p.ctx.Logger != nil {
				p.ctx.Logger.Debug("[Processor] Shutdown")
			}
			close(p.reqChan)
			for req := range p.reqChan {
				// Respond to each pending request with cancellation
				resp := DrawResponse{RequestID: req.RequestID, Item: "", Err: types.ErrShutingDown}
				if req.Callback != nil {
					req.Callback(resp)
				}
			}
			// Final flush/commit on shutdown
			p.Flush()
			return
		}
	}
}

func (p *Processor) Flush() error {
	if p.stagedDraws < 0 {
		return nil
	}

	flushErr := p.ctx.WAL.Flush()
	if flushErr == nil {
		p.pool.CommitDraw()
		if p.ctx.Logger != nil {
			p.ctx.Logger.Debug(fmt.Sprintf("[Processor] WAL Flush and Commit - %d", p.stagedDraws))
		}
	} else {
		p.pool.RevertDraw()
		if p.ctx.Logger != nil {
			p.ctx.Logger.Debug(fmt.Sprintf("[Processor] WAL Flush and Revert - %d", p.stagedDraws))
		}

	}
	p.stagedDraws = 0
	return flushErr
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
