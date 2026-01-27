package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds environment and runtime configuration.
type Config struct {
	Mode     string `yaml:"mode"`
	APIPort  int    `yaml:"api_port"`
	WebPort  int    `yaml:"web_port"`
	DBPath   string `yaml:"db_path"`
}

// Load reads YAML config from path, applying defaults where fields are missing.
func Load(path string) (*Config, error) {
	cfg := &Config{
		Mode:    "live",
		APIPort: 5000,
		WebPort: 5001,
		DBPath:  "hft.db",
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Re-apply defaults if YAML omits a field (zero values).
	if cfg.Mode == "" {
		cfg.Mode = "live"
	}
	if cfg.APIPort == 0 {
		cfg.APIPort = 5000
	}
	if cfg.WebPort == 0 {
		cfg.WebPort = 5001
	}
	if cfg.DBPath == "" {
		cfg.DBPath = "hft.db"
	}

	return cfg, nil
}
