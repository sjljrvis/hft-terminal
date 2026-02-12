package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FyersConfig holds configuration for the Fyers broker.
type FyersConfig struct {
	AppID       string `yaml:"app_id"`
	AppSecret   string `yaml:"app_secret"`
	RedirectURI string `yaml:"redirect_uri"`
	Pin         string `yaml:"pin"`
}

// BrokerConfig represents a single broker entry in configuration.
type BrokerConfig struct {
	Fyers FyersConfig `yaml:"fyers"`
}

// ClockConfig controls market session times (useful for testing off-market hours).
// Times are "time of day" strings like "09:15" or "9:15 AM" or "11PM".
type ClockConfig struct {
	Location   string `yaml:"location"`   // e.g. "Asia/Kolkata"
	Start      string `yaml:"start"`      // e.g. "09:15"
	End        string `yaml:"end"`        // e.g. "15:30"
	Deactivate string `yaml:"deactivate"` // optional; if empty defaults to end + 10 minutes
}

// Config holds environment and runtime configuration.
type Config struct {
	Mode    string         `yaml:"mode"`
	APIPort int            `yaml:"api_port"`
	WebPort int            `yaml:"web_port"`
	DBPath  string         `yaml:"db_path"`
	Broker  []BrokerConfig `yaml:"broker"`
	Clock   ClockConfig    `yaml:"clock"`
}

var GlobalConfig *Config

// Load reads YAML config from path, applying defaults where fields are missing.
func Load(path string) (*Config, error) {
	cfg := &Config{
		Mode:    "live",
		APIPort: 5000,
		WebPort: 5001,
		DBPath:  "hft.db",
		Clock: ClockConfig{
			Location:   "Asia/Kolkata",
			Start:      "09:15",
			End:        "15:30",
			Deactivate: "",
		},
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
	if cfg.Clock.Location == "" {
		cfg.Clock.Location = "Asia/Kolkata"
	}
	if cfg.Clock.Start == "" {
		cfg.Clock.Start = "09:15"
	}
	if cfg.Clock.End == "" {
		cfg.Clock.End = "15:30"
	}
	// If Deactivate is empty, clock package will default it to End + 10 minutes.

	GlobalConfig = cfg
	return cfg, nil
}
