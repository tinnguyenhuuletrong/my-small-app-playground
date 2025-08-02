package config

import (
	"encoding/json"
	"os"
)

func LoadConfig(path string, v interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	return json.NewDecoder(file).Decode(v)
}
