package processing

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type DrawRequest struct {
	RequestID    uint64
	ResponseChan chan DrawResponse
	Callback     func(DrawResponse)
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
	reqChan         chan *DrawRequest
	stopChan        chan struct{}
	wg              sync.WaitGroup
	flushAfterNDraw int
	stagedDraws     int
	drawRequestPool sync.Pool
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
		reqChan:         make(chan *DrawRequest, bufSize),
		stopChan:        make(chan struct{}),
		flushAfterNDraw: n,
		drawRequestPool: sync.Pool{
			New: func() interface{} {
				return &DrawRequest{
					ResponseChan: make(chan DrawResponse, 1),
				}
			},
		},
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
			logItem := types.WalLogDrawItem{
				WalLogItem: types.WalLogItem{
					Type: types.LogTypeDraw,
				},
				RequestID: req.RequestID,
				Success:   err == nil,
			}

			if logItem.Success {
				logItem.ItemID = item
			} else {
				if err == types.ErrEmptyRewardPool {
					logItem.Error = types.ErrorPoolEmpty
				}
				// NOTE: We can map more error types here in the future
			}
			walErr = p.ctx.WAL.LogDraw(logItem)
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

			if req.ResponseChan != nil {
				req.ResponseChan <- resp
				p.drawRequestPool.Put(req)
			} else if req.Callback != nil {
				req.Callback(resp)
			}
		case <-p.stopChan:
			// Graceful shutdown: stop receiving requests, cancel pending, then flush
			// Drain reqChan and cancel all pending requests
			if p.ctx.Utils.GetLogger() != nil {
				p.ctx.Utils.GetLogger().Debug("[Processor] Shutdown")
			}
			close(p.reqChan)
			for req := range p.reqChan {
				resp := DrawResponse{RequestID: req.RequestID, Item: "", Err: types.ErrShutingDown}
				if req.ResponseChan != nil {
					req.ResponseChan <- resp
					p.drawRequestPool.Put(req)
				} else if req.Callback != nil {
					req.Callback(resp)
				}
			}
			// Final flush/commit on shutdown
			p.Flush()

			// Close wall
			p.ctx.WAL.Close()
			return
		}
	}
}

func (p *Processor) Flush() error {
	if p.stagedDraws <= 0 {
		return nil
	}

	flushErr := p.ctx.WAL.Flush()
	shouldCreateSnapshot := false
	if flushErr == types.ErrWALFull {
		// WAL is full, let's rotate it.
		if logger := p.ctx.Utils.GetLogger(); logger != nil {
			logger.Info("WAL is full, starting rotation and snapshot process.")
		}

		// 1. Rotate the WAL file.
		rotatedPath := p.ctx.Utils.GenRotatedWALPath()
		if rotatedPath != nil {
			if err := p.ctx.WAL.Rotate(*rotatedPath); err != nil {
				if logger := p.ctx.Utils.GetLogger(); logger != nil {
					logger.Error("Failed to rotate WAL. Reverting draws.", "error", err)
				}
				p.pool.RevertDraw()
				p.stagedDraws = 0
				return err // This is a critical failure.
			}
		} else {
			if logger := p.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("Wall is full, rotatedPath not set. Noting to do. Stop here")
			}
			panic(1)
		}

		// 2. The buffer is still holding the data that failed to write.
		//    Let's try flushing it again to the new, empty WAL.
		flushErr = p.ctx.WAL.Flush()
		if flushErr != nil {
			if logger := p.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("Failed 2nd flush.", "error", flushErr)
			}
		}

		// After a rotation, we create a snapshot.
		if flushErr == nil {
			shouldCreateSnapshot = true
		}
	}

	// Final commit/revert logic based on the outcome.
	if flushErr == nil {
		p.pool.CommitDraw()
		if logger := p.ctx.Utils.GetLogger(); logger != nil {
			logger.Debug(fmt.Sprintf("[Processor] WAL Flush and Commit - %d", p.stagedDraws))
		}
	} else {
		p.pool.RevertDraw()
		if logger := p.ctx.Utils.GetLogger(); logger != nil {
			logger.Error("[Processor] WAL Flush failed, reverting draws.", "error", flushErr)
		}
	}

	// Create snapshot
	if shouldCreateSnapshot {
		p.snapshot()
	}

	p.stagedDraws = 0
	return flushErr
}

func (p *Processor) snapshot() error {
	snapshotPath := p.ctx.Utils.GenSnapshotPath()
	if snapshotPath != nil {
		if logger := p.ctx.Utils.GetLogger(); logger != nil {
			logger.Info("Creating snapshot.", "path", *snapshotPath)
		}
		// The snapshot should represent the state of the pool *before* the draws that are now in the new WAL.
		// This is the correct state, as it represents the end of the previous (now archived) WAL.
		if err := p.pool.SaveSnapshot(*snapshotPath); err != nil {
			// Log snapshot error but don't fail the whole operation,
			// as the draws have been successfully logged to the new WAL.
			if logger := p.ctx.Utils.GetLogger(); logger != nil {
				logger.Error("Failed to create snapshot after WAL rotation.", "error", err)
			}
			return err
		}
	}
	return nil
}

func (p *Processor) Draw() <-chan DrawResponse {
	req := p.drawRequestPool.Get().(*DrawRequest)
	req.RequestID = atomic.AddUint64(&p.requestID, 1)
	p.reqChan <- req
	return req.ResponseChan
}

func (p *Processor) DrawWithCallback(callback func(DrawResponse)) uint64 {
	reqID := atomic.AddUint64(&p.requestID, 1)
	p.reqChan <- &DrawRequest{RequestID: reqID, Callback: callback}
	return reqID
}

func (p *Processor) Stop() {
	close(p.stopChan)
	p.wg.Wait()
}
