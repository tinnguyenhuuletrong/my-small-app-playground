package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/cmd/cli/tui"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/actor"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/config"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/recovery"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/rewardpool"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/utils"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal"
	walformatter "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/formatter"
	walstorage "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/wal/storage"
	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/walstream"
)

func main() {
	// Load config
	c := &config.ConfigImpl{}
	cfg, err := c.LoadYAML("samples/config.yaml")
	if err != nil {
		log.Fatalf("LoadConfig failed: %v", err)
	}

	// Setup paths
	baseDir := "."
	tmpDir := baseDir + "/" + cfg.WorkingDir
	snapshotPath := tmpDir + "/snapshot.json"
	walPath := tmpDir + "/wal.log"

	// Create tmpDir if not exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		os.MkdirAll(tmpDir, 0755)
	}

	logChan := make(chan string, 100)
	writer := &tui.ChannelWriter{Ch: logChan}

	utils := utils.NewDefaultUtils(tmpDir, tmpDir, slog.LevelDebug, writer)

	var walFormatter types.LogFormatter
	switch cfg.WAL.Formatter {
	case "json":
		walFormatter = walformatter.NewJSONFormatter()
	case "string_line":
		walFormatter = walformatter.NewStringLineFormatter()
	default:
		log.Fatalf("Unsupported WAL formatter: %s", cfg.WAL.Formatter)
	}

	// Create a pool from the config
	initialPool := rewardpool.CreatePoolFromConfig(cfg.Pool)

	pool, lastRequestID, err := recovery.RecoverPoolFromConfig(snapshotPath, walPath, initialPool, walFormatter, utils)
	if err != nil {
		fmt.Println("Recovery failed:", err)
		os.Exit(1)
	}

	fileStorage, err := walstorage.NewFileStorage(walPath, walstorage.FileStorageOpt{
		SizeFileInBytes: cfg.WAL.MaxFileSizeKB * 1024, // From KB to Bytes
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

	walStreamer := walstream.NewNoOpStreamer()

	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{
		FlushAfterNDraw:   cfg.WAL.FlushAfterNDraw,
		RequestBufferSize: cfg.WAL.MaxRequestBuffer,
		LastRequestID:     lastRequestID,
		WALStreamer:       walStreamer,
	})
	if err != nil {
		fmt.Println("System startup error:", err)
		return
	}
	sys.SetRequestID(lastRequestID)

	p := tea.NewProgram(tui.NewModel(sys, logChan))
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, there's been an error: %v", err)
		os.Exit(1)
	}

	fmt.Println("Shutdown complete.")
}
