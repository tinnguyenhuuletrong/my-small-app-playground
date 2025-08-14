package main

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

func BenchmarkProcessorDrawChannel(b *testing.B) {
	ctx := &types.Context{Utils: &utils.MockUtils{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 1.0},
		},
	)
	w := &utils.MockWAL{}
	ctx.WAL = w

	opt := &processing.ProcessorOptional{RequestBufferSize: b.N}
	p := processing.NewProcessor(ctx, pool, opt)

	b.ResetTimer()

	resChans := make([]<-chan processing.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = p.Draw()
	}

	for _, ch := range resChans {
		<-ch
	}

	p.Stop()
}

func BenchmarkActorDrawChannel(b *testing.B) {
	ctx := &types.Context{Utils: &utils.MockUtils{}}
	pool := rewardpool.NewPool(
		[]types.PoolReward{
			{ItemID: "gold", Quantity: b.N, Probability: 1.0},
		},
	)
	w := &utils.MockWAL{}
	ctx.WAL = w

	opt := &actor.SystemOptional{RequestBufferSize: b.N}
	sys := actor.NewSystem(ctx, pool, opt)

	b.ResetTimer()

	resChans := make([]<-chan actor.DrawResponse, b.N)
	for i := 0; i < b.N; i++ {
		resChans[i] = sys.Draw()
	}

	for _, ch := range resChans {
		<-ch
	}

	sys.Stop()
}
