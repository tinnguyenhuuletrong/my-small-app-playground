package main

import (
	"context"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/my-go-p2p/internal"
)

func main() {
	ctx := context.Background()
	internal.StartBroadCast(ctx)
}
