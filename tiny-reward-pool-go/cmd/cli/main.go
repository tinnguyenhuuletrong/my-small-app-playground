package main

import (
	"fmt"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
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

	// Example: Draw first item
	if len(pool.Catalog) > 0 && pool.Catalog[0].Quantity > 0 {
		pool.Catalog[0].Quantity--
		w.LogDraw(1, pool.Catalog[0].ItemID, true)
		fmt.Printf("Drew %s, remaining: %d\n", pool.Catalog[0].ItemID, pool.Catalog[0].Quantity)
	} else {
		w.LogDraw(1, "", false)
		fmt.Println("Draw failed: pool empty")
	}
}
