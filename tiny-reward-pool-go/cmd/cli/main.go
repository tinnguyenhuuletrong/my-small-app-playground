package main

import (
	"context"
	"flag"
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
	var configPath string
	flag.StringVar(&configPath, "config", "", "path to the config.yaml file")
	flag.Parse()

	if configPath == "" {
		fmt.Println("Error: config file path is required.")
		flag.Usage()
		os.Exit(1)
	}

	c := &config.ConfigImpl{}
	cfg, err := c.LoadYAML(configPath)
	if err != nil {
		log.Fatalf("LoadConfig failed: %v", err)
	}

	for {
		sys, writer, err := setup(cfg)
		if err != nil {
			log.Fatalf("Setup failed: %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		if cfg.GRPC.Enabled {
			go func() {
				log.Printf("server listening at %v", cfg.GRPC.ListenAddress)
				if err := rewardpool_grpc_service.ListenAndServe(ctx, sys, cfg.GRPC.ListenAddress); err != nil {
					log.Fatalf("failed to serve: %v", err)
				}
			}()
		}

		m := tui.NewModel(sys, writer.GetReaderChan())
		p := tea.NewProgram(m)
		finalModel, err := p.Run()

		sys.Stop()
		fmt.Println("Shutdown complete.")
		cancel()

		writer.Close()

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

func setup(cfg config.YAMLConfig) (*actor.System, *tui.ChannelWriter, error) {
	// Setup paths
	baseDir := "."
	tmpDir := baseDir + "/" + cfg.WorkingDir

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

	pool, lastRequestID, lastWalPath, err := recovery.RecoverPoolFromConfig(initialPool, walFormatter, utils)
	if err != nil {
		return nil, nil, fmt.Errorf("recovery failed: %w", err)
	}

	var w types.WAL
	var seqNo uint64
	if lastWalPath == "" {
		var newWalPath string
		newWalPath, seqNo, err = utils.GenNextWALPath()
		if err != nil {
			return nil, nil, fmt.Errorf("error generating new WAL path: %w", err)
		}
		lastWalPath = newWalPath
	}

	fileStorage, err := walstorage.NewFileMMapStorage(lastWalPath, seqNo, walstorage.FileMMapStorageOps{
		MMapFileSizeInBytes: int64(cfg.WAL.MaxFileSizeKB * 1024), // From KB to Bytes
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error creating file storage: %w", err)
	}
	w, err = wal.NewWAL(lastWalPath, seqNo, walFormatter, fileStorage)
	if err != nil {
		return nil, nil, fmt.Errorf("error opening WAL: %w", err)
	}

	ctx := &types.Context{
		WAL:   w,
		Utils: utils,
	}

	walStreamer := walstream.NewNoOpStreamer()

	walFactory := func(path string, seqNo uint64) (types.WAL, error) {
		fileStorage, err := walstorage.NewFileMMapStorage(path, seqNo, walstorage.FileMMapStorageOps{
			MMapFileSizeInBytes: int64(cfg.WAL.MaxFileSizeKB * 1024), // From KB to Bytes
		})
		if err != nil {
			return nil, fmt.Errorf("error creating file storage: %w", err)
		}
		return wal.NewWAL(path, seqNo, walFormatter, fileStorage)
	}

	sys, err := actor.NewSystem(ctx, pool, &actor.SystemOptional{
		FlushAfterNDraw:   cfg.WAL.FlushAfterNDraw,
		RequestBufferSize: cfg.WAL.MaxRequestBuffer,
		LastRequestID:     lastRequestID,
		WALStreamer:       walStreamer,
		WALFactory:        walFactory,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("system startup error: %w", err)
	}
	sys.SetRequestID(lastRequestID)

	utils.GetLogger().Debug(fmt.Sprintf("Config: %+v", cfg))
	return sys, writer, nil
}