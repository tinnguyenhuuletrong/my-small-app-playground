package main

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
)

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
	sys, err := actor.NewSystem(ctx, pool, opt)
	if err != nil {
		b.Error(err)
	}

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
