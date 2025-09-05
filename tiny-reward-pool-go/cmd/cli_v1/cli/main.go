package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/walstream"
)

func main() {
	// baseDir := "..//.."
	baseDir := "."
	defaultConfigPath := baseDir + "/samples/config.json"
	tmpDir := baseDir + "/tmp"

	utils := utils.NewDefaultUtils(tmpDir, tmpDir, slog.LevelDebug, nil)

	// walFormatter := walformatter.NewJSONFormatter()
	walFormatter := walformatter.NewStringLineFormatter()
	pool, lastRequestID, lastWalPath, err := recovery.RecoverPool(defaultConfigPath, walFormatter, utils)
	if err != nil {
		fmt.Println("Recovery failed:", err)
		os.Exit(1)
	}

	var w types.WAL
	var seqNo uint64
	if lastWalPath == "" {
		var newWalPath string
		newWalPath, seqNo, err = utils.GenNextWALPath()
		if err != nil {
			fmt.Println("Error generating new WAL path:", err)
			os.Exit(1)
		}
		lastWalPath = newWalPath
	}

	// fileStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
	// 	MMapFileSizeInBytes: 1024 * 0.5, // 0.5 Kb
	// })
	fileStorage, err := walstorage.NewFileStorage(lastWalPath, seqNo, walstorage.FileStorageOpt{
		// SizeFileInBytes: 1024 * 1024 * 0.5, // 0.5 MB
		SizeFileInBytes: int(math.Round(1024 * 0.2)), // 0.5 Kb
	})
	if err != nil {
		fmt.Println("Error creating file storage:", err)
		os.Exit(1)
	}
	w, err = wal.NewWAL(lastWalPath, seqNo, walFormatter, fileStorage)
	if err != nil {
		fmt.Println("Error opening WAL:", err)
		os.Exit(1)
	}

	ctx := &types.Context{
		WAL:   w,
		Utils: utils,
	}

	// Check for --stream-wal flag
	streamWAL := false
	for _, arg := range os.Args {
		if arg == "--stream-wal" {
			streamWAL = true
			break
		}
	}

	var walStreamer walstream.WALStreamer
	if streamWAL {
		fmt.Println("WAL streaming is enabled.")
		walStreamer = walstream.NewLogStreamer(utils.GetLogger())
	} else {
		walStreamer = walstream.NewNoOpStreamer()
	}

	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{
		FlushAfterNDraw: 5,
		LastRequestID:   lastRequestID,
		WALStreamer:     walStreamer,
	})
	if err != nil {
		fmt.Println("System startup error:", err)
		return
	}
	sys.SetRequestID(lastRequestID)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("CLI Controls:")
	fmt.Println("  - Press '1' to add 10 gold items.")
	fmt.Println("  - Press '2' to toggle silver probability between 10 and 90.")
	fmt.Println("  - Press Ctrl+C or send SIGTERM to exit.")
	fmt.Println("-------------------------------------------------")
	fmt.Println("[Pool state] ", pool.State())

	drawLock := make(chan struct{}, 1) // Used to lock draw requests
	drawLock <- struct{}{}

	go func() {
		for {
			<-drawLock
			resp := <-sys.Draw()
			if resp.Err == nil {
				fmt.Printf("[Request %d] Drew %s\n", resp.RequestID, resp.Item)
			} else {
				fmt.Printf("[Request %d] Draw failed: %s \n", resp.RequestID, resp.Err)
			}
			drawLock <- struct{}{}
			time.Sleep(1 * time.Second)
		}
	}()

	// Goroutine to handle user input
	silverProbToggle := false
	go func() {
		reader := bufio.NewReader(os.Stdin)
		for {
			char, _, err := reader.ReadRune()
			if err != nil {
				fmt.Println("Error reading input:", err)
				return
			}

			switch char {
			case '1':
				fmt.Println("\n--- Adding 10 gold... ---")
				var currentGold types.PoolReward
				for _, item := range sys.State() {
					if item.ItemID == "gold" {
						currentGold = item
						break
					}
				}
				err := sys.UpdateItem("gold", currentGold.Quantity+10, currentGold.Probability)
				if err != nil {
					fmt.Printf("Failed to update gold: %v\n", err)
				} else {
					fmt.Println("--- Gold updated. New pool state: ---")
					fmt.Println(sys.State())
					fmt.Println("-----------------------------------------")
				}

			case '2':
				fmt.Println("\n--- Toggling silver probability... ---")
				var currentSilver types.PoolReward
				for _, item := range sys.State() {
					if item.ItemID == "silver" {
						currentSilver = item
						break
					}
				}

				var newProb int64
				if silverProbToggle {
					newProb = 10
				} else {
					newProb = 90
				}
				silverProbToggle = !silverProbToggle

				err := sys.UpdateItem("silver", currentSilver.Quantity, newProb)
				if err != nil {
					fmt.Printf("Failed to update silver: %v\n", err)
				} else {
					fmt.Println("--- Silver updated. New pool state: ---")
					fmt.Println(sys.State())
					fmt.Println("-----------------------------------------")
				}
			}
		}
	}()

	<-sigChan
	fmt.Println("Shutting down gracefully...")
	<-drawLock

	sys.Stop()

	fmt.Println("[Pool state] ", pool.State())
	fmt.Println("Shutdown complete.")
}