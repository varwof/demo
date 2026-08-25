package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type AppConfig struct {
	Listen string `json:"listen"`
	DSN    string `json:"dsn"`
}

func LoadAppConfig(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Listen == "" {
		cfg.Listen = ":9393"
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("dsn required")
	}
	return &cfg, nil
}
