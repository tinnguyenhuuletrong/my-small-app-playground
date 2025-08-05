package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/processing"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
)

func main() {
	snapshotPath := "./tmp/pool_snapshot.json"
	walPath := "./tmp/wal.log"

	pool, err := recovery.RecoverPool(snapshotPath, walPath, "./samples/config.json")
	if err != nil {
		fmt.Println("Recovery failed:", err)
		os.Exit(1)
	}

	w, err := wal.NewWAL(walPath)
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

	fmt.Println("[Pool state] ", pool)

	fmt.Println("Press Ctrl+C or send SIGTERM to exit.")

	drawLock := make(chan struct{}, 1) // Used to lock draw requests
	drawLock <- struct{}{}             // Initially unlocked

	go func() {
		for {
			<-drawLock // Wait for unlock
			reqID := proc.Draw(func(resp processing.DrawResponse) {
				if resp.Item != nil {
					fmt.Printf("[Request %d] Drew %s, remaining: %d\n", resp.RequestID, resp.Item.ItemID, resp.Item.Quantity)
				} else {
					fmt.Printf("[Request %d] Draw failed: pool empty\n", resp.RequestID)
				}
			})
			fmt.Printf("Draw requested, requestID: %d\n", reqID)
			time.Sleep(2 * time.Second)
			drawLock <- struct{}{} // Unlock for next draw
		}
	}()

	go func() {
		for {
			time.Sleep(5 * time.Second)
			// Lock draw requests
			<-drawLock
			w.Flush()
			fmt.Println("[Pool state] ", pool)
			fmt.Println("Saving pool snapshot...")
			if err := pool.SaveSnapshot(snapshotPath); err != nil {
				fmt.Println("Error saving snapshot:", err)
			} else {
				fmt.Println("Snapshot saved.")
			}
			fmt.Println("Rotating WAL file...")
			w.Close()
			os.Remove(walPath)
			w, err = wal.NewWAL(walPath)
			if err != nil {
				fmt.Println("Error creating new WAL:", err)
				os.Exit(1)
			}
			ctx.WAL = w
			fmt.Println("WAL rotated.")
			drawLock <- struct{}{} // Unlock draw requests
		}
	}()

	<-sigChan
	fmt.Println("Shutting down gracefully...")
	proc.Stop()
	w.Flush()
	w.Close()
	fmt.Println("Shutdown complete.")
}
