package main

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func main() {
	pool, err := rewardpool.LoadPool("./samples/config.json")
	if err != nil {
		fmt.Println("Error loading pool:", err)
		os.Exit(1)
	}
	w, err := wal.NewWAL("./tmp/wal.log")
	if err != nil {
		fmt.Println("Error opening WAL:", err)
		os.Exit(1)
	}
	defer w.Close()

	ctx := &types.Context{
		WAL:   w,
		Utils: &utils.UtilsImpl{},
	}

	result, err := pool.Draw(ctx)
	if err != nil {
		fmt.Println("Draw error:", err)
		os.Exit(1)
	}
	if result != nil {
		fmt.Printf("Drew %s, remaining: %d\n", result.ItemID, result.Quantity)
	} else {
		fmt.Println("Draw failed: pool empty")
	}
}
