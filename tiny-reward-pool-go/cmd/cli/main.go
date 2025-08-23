package main

import (
	"context"
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
	rewardpool_grpc_service "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/pkg/rewardpool-grpc-service"
)

func main() {
	c := &config.ConfigImpl{}
	cfg, err := c.LoadYAML("samples/config.yaml")
	if err != nil {
		log.Fatalf("LoadConfig failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for {
		sys, logChan, err := setup(cfg)
		if err != nil {
			log.Fatalf("Setup failed: %v", err)
		}

		if cfg.GRPC.Enabled {
			go func() {
				log.Printf("server listening at %v", cfg.GRPC.ListenAddress)
				if err := rewardpool_grpc_service.ListenAndServe(ctx, sys, cfg.GRPC.ListenAddress); err != nil {
					log.Fatalf("failed to serve: %v", err)
				}
			}()
		}

		m := tui.NewModel(sys, logChan)
		p := tea.NewProgram(m)
		finalModel, err := p.Run()

		sys.Stop()
		fmt.Println("Shutdown complete.")
		cancel() // Cancel the context to stop the gRPC server

		// close logChan
		close(logChan)
		for value := range logChan {
			fmt.Println(value)
		}

		if err != nil {
			log.Printf("TUI error: %v", err)
			// if error -> not reload
			break
		}

		if finalModel.(tui.Model).ShouldReload {
			fmt.Println("Reloading...")
			continue
		}

		break
	}
}

func setup(cfg config.YAMLConfig) (*actor.System, chan string, error) {
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
		return nil, nil, fmt.Errorf("unsupported WAL formatter: %s", cfg.WAL.Formatter)
	}

	// Create a pool from the config
	initialPool := rewardpool.CreatePoolFromConfig(cfg.Pool)

	pool, lastRequestID, err := recovery.RecoverPoolFromConfig(snapshotPath, walPath, initialPool, walFormatter, utils)
	if err != nil {
		return nil, nil, fmt.Errorf("recovery failed: %w", err)
	}

	fileStorage, err := walstorage.NewFileStorage(walPath, walstorage.FileStorageOpt{
		SizeFileInBytes: cfg.WAL.MaxFileSizeKB * 1024, // From KB to Bytes
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error creating file storage: %w", err)
	}
	w, err := wal.NewWAL(walPath, walFormatter, fileStorage)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening WAL: %w", err)
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
		return nil, nil, fmt.Errorf("system startup error: %w", err)
	}
	sys.SetRequestID(lastRequestID)

	utils.GetLogger().Debug(fmt.Sprintf("Config: %+v", cfg))
	return sys, logChan, nil
}