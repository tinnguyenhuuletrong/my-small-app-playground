package config

import (
	"encoding/json"
	"os"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/types"
	"gopkg.in/yaml.v3"
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

func (c *ConfigImpl) LoadYAML(path string) (YAMLConfig, error) {
	file, err := os.Open(path)
	if err != nil {
		return YAMLConfig{}, err
	}
	defer file.Close()
	var cfg YAMLConfig
	err = yaml.NewDecoder(file).Decode(&cfg)
	return cfg, err
}
