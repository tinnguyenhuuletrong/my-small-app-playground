package config

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
)

type ConfigImpl struct{}

func (c *ConfigImpl) LoadConfig(path string) (types.ConfigPool, error) {
	file, err := os.Open(path)
	if err != nil {
		return types.ConfigPool{}, err
	}
	defer file.Close()
	var cfg types.ConfigPool
	err = json.NewDecoder(file).Decode(&cfg)
	return cfg, err
}
