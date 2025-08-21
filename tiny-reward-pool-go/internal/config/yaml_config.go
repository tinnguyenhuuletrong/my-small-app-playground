package config

import "github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"

// YAMLConfig represents the application's configuration.
type YAMLConfig struct {
	WorkingDir string        `yaml:"working_dir"`
	Pool       types.ConfigPool `yaml:"pool"`
	WAL        YAMLConfigWAL  `yaml:"wal"`
}

// YAMLConfigWAL represents the configuration for the WAL.
type YAMLConfigWAL struct {
	MaxFileSize   int    `yaml:"max_file_size"`
	MaxBufferSize int    `yaml:"max_buffer_size"`
	Formatter     string `yaml:"formatter"`
}
