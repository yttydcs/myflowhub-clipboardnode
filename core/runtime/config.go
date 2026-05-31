package runtime

import (
	"fmt"
	"strings"
)

const (
	DefaultMaxInlineBytes = 65536
	defaultRecentLimit    = 128
	defaultSuppressLimit  = 32
)

type Config struct {
	Enabled        bool   `json:"enabled"`
	Topic          string `json:"topic"`
	MaxInlineBytes int    `json:"max_inline_bytes"`
	DeviceLabel    string `json:"device_label,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		Enabled:        false,
		MaxInlineBytes: DefaultMaxInlineBytes,
	}
}

func NormalizeConfig(cfg Config) (Config, error) {
	cfg.Topic = strings.TrimSpace(cfg.Topic)
	cfg.DeviceLabel = strings.TrimSpace(cfg.DeviceLabel)
	if cfg.MaxInlineBytes == 0 {
		cfg.MaxInlineBytes = DefaultMaxInlineBytes
	}
	if cfg.MaxInlineBytes < 0 {
		return Config{}, fmt.Errorf("max_inline_bytes must be positive")
	}
	if cfg.Enabled && cfg.Topic == "" {
		return Config{}, fmt.Errorf("topic is required when clipboard sync is enabled")
	}
	return cfg, nil
}
