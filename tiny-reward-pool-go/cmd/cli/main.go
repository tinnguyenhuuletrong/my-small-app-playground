package main

import (
	"fmt"
	"log"

	"github.com/tinnguyenhuuletrong/my-small-app-playground/tiny-reward-pool-go/internal/config"
)

func main() {
	c := &config.ConfigImpl{}
	cfg, err := c.LoadYAML("samples/config.yaml")
	if err != nil {
		log.Fatalf("LoadConfig failed: %v", err)
	}

	fmt.Printf("%+v\n", cfg)
}

