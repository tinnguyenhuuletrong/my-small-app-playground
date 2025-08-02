package config_test

import (
	"testing"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/config"
)

func TestLoadConfig(t *testing.T) {
	c := &config.ConfigImpl{}
	_, err := c.LoadConfig("../../samples/config.json")
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
}
