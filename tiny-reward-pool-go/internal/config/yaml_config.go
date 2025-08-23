package config

import "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

// YAMLConfig represents the application's configuration.
type YAMLConfig struct {
	WorkingDir string           `yaml:"working_dir"`
	Pool       types.ConfigPool `yaml:"pool"`
	WAL        YAMLConfigWAL    `yaml:"wal"`
	GRPC       YAMLConfigGRPC   `yaml:"grpc"`
}

// YAMLConfigWAL represents the configuration for the WAL.
type YAMLConfigWAL struct {
	MaxFileSizeKB    int    `yaml:"max_file_size_kb"`
	MaxRequestBuffer int    `yaml:"max_request_buffer_size"`
	Formatter        string `yaml:"formatter"`
	FlushAfterNDraw  int    `yaml:"flush_after_n_draw"`
}

// YAMLConfigGRPC represents the configuration for the gRPC service.
type YAMLConfigGRPC struct {
	Enabled       bool   `yaml:"enabled"`
	ListenAddress string `yaml:"listen_address"`
}
