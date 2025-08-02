package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
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
	proc := processing.NewProcessor(ctx, pool)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("Press Ctrl+C or send SIGTERM to exit.")

	go func() {
		for {
			reqID := proc.Draw(func(resp processing.DrawResponse) {
				if resp.Item != nil {
					fmt.Printf("[Request %d] Drew %s, remaining: %d\n", resp.RequestID, resp.Item.ItemID, resp.Item.Quantity)
				} else {
					fmt.Printf("[Request %d] Draw failed: pool empty\n", resp.RequestID)
				}
			})
			fmt.Printf("Draw requested, requestID: %d\n", reqID)
			time.Sleep(2 * time.Second)
		}
	}()

	<-sigChan
	fmt.Println("Shutting down gracefully...")
	proc.Stop()
	fmt.Println("Shutdown complete.")
}
