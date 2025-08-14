package main

import (
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
)

func main() {
	// baseDir := "../.."
	baseDir := "."
	defaultConfigPath := baseDir + "/samples/config.json"
	tmpDir := baseDir + "/tmp"
	snapshotPath := tmpDir + "/snapshot.json"
	walPath := tmpDir + "/wal.log"

	utils := utils.NewDefaultUtils(tmpDir, tmpDir, slog.LevelDebug)

	// walFormatter := walformatter.NewJSONFormatter()
	walFormatter := walformatter.NewStringLineFormatter()
	pool, err := recovery.RecoverPool(snapshotPath, walPath, defaultConfigPath, walFormatter, utils)
	if err != nil {
		fmt.Println("Recovery failed:", err)
		os.Exit(1)
	}

	// fileStorage, err := walstorage.NewFileMMapStorage(walPath, walstorage.FileMMapStorageOps{
	// 	MMapFileSizeInBytes: 1024 * 0.5, // 0.5 Kb
	// })
	fileStorage, err := walstorage.NewFileStorage(walPath, walstorage.FileStorageOpt{
		// SizeFileInBytes: 1024 * 1024 * 0.5, // 0.5 MB
		SizeFileInBytes: int(math.Round(1024 * 0.2)), // 0.5 Kb
	})
	if err != nil {
		fmt.Println("Error creating file storage:", err)
		os.Exit(1)
	}
	w, err := wal.NewWAL(walPath, walFormatter, fileStorage)
	if err != nil {
		fmt.Println("Error opening WAL:", err)
		os.Exit(1)
	}

	ctx := &types.Context{
		WAL:   w,
		Utils: utils,
	}
	sys := actor.NewSystem(ctx, pool, &actor.SystemOptional{
		FlushAfterNDraw: 5,
	})

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("[Pool state] ", pool.State())
	fmt.Println("Press Ctrl+C or send SIGTERM to exit.")

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

	<-sigChan
	fmt.Println("Shutting down gracefully...")
	<-drawLock

	sys.Stop()

	fmt.Println("[Pool state] ", pool.State())
	fmt.Println("Shutdown complete.")
}
