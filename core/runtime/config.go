package runtime

import (
	"fmt"
	"strings"
)

const (
	DefaultParentEndpoint    = "127.0.0.1:9000"
	DefaultMaxInlineBytes    = 65536
	DefaultDeviceID          = "local-device"
	DefaultHistoryLimit      = 256
	MaxHistoryLimit          = 5000
	HistoryRetentionNone     = "none"
	HistoryRetentionMetadata = "metadata"
	HistoryRetentionBody     = "body"
	defaultRecentLimit       = 128
	defaultSuppressLimit     = 32
)

type Config struct {
	Enabled          bool   `json:"enabled"`
	ParentEndpoint   string `json:"parent_endpoint"`
	Topic            string `json:"topic"`
	MaxInlineBytes   int    `json:"max_inline_bytes"`
	DeviceID         string `json:"device_id,omitempty"`
	DisplayName      string `json:"display_name,omitempty"`
	DeviceLabel      string `json:"device_label,omitempty"`
	AutoWatch        bool   `json:"auto_watch"`
	AutoApply        bool   `json:"auto_apply"`
	HistoryRetention string `json:"history_retention,omitempty"`
	HistoryLimit     int    `json:"history_limit,omitempty"`
	TransferProvider string `json:"transfer_provider,omitempty"`
	TransferRef      string `json:"transfer_ref,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		ParentEndpoint:   DefaultParentEndpoint,
		Enabled:          false,
		MaxInlineBytes:   DefaultMaxInlineBytes,
		DeviceID:         DefaultDeviceID,
		DisplayName:      DefaultDeviceID,
		DeviceLabel:      DefaultDeviceID,
		AutoWatch:        false,
		AutoApply:        false,
		HistoryRetention: HistoryRetentionBody,
		HistoryLimit:     DefaultHistoryLimit,
	}
}

func NormalizeConfig(cfg Config) (Config, error) {
	cfg.ParentEndpoint = strings.TrimSpace(cfg.ParentEndpoint)
	cfg.Topic = strings.TrimSpace(cfg.Topic)
	cfg.DeviceID = strings.TrimSpace(cfg.DeviceID)
	cfg.DisplayName = strings.TrimSpace(cfg.DisplayName)
	cfg.DeviceLabel = strings.TrimSpace(cfg.DeviceLabel)
	cfg.HistoryRetention = strings.TrimSpace(cfg.HistoryRetention)
	cfg.TransferProvider = strings.TrimSpace(cfg.TransferProvider)
	cfg.TransferRef = strings.TrimSpace(cfg.TransferRef)
	if cfg.DeviceID == "" {
		cfg.DeviceID = cfg.DeviceLabel
	}
	if cfg.DeviceID == "" {
		cfg.DeviceID = DefaultDeviceID
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = cfg.DeviceLabel
	}
	if cfg.DisplayName == "" {
		cfg.DisplayName = cfg.DeviceID
	}
	cfg.DeviceLabel = cfg.DisplayName
	if cfg.HistoryRetention == "" {
		cfg.HistoryRetention = HistoryRetentionBody
	}
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = DefaultHistoryLimit
	}
	if cfg.MaxInlineBytes == 0 {
		cfg.MaxInlineBytes = DefaultMaxInlineBytes
	}
	if cfg.ParentEndpoint == "" {
		cfg.ParentEndpoint = DefaultParentEndpoint
	}
	if cfg.MaxInlineBytes < 0 {
		return Config{}, fmt.Errorf("max_inline_bytes must be positive")
	}
	if cfg.HistoryLimit < 0 {
		return Config{}, fmt.Errorf("history_limit must be positive")
	}
	if cfg.HistoryLimit > MaxHistoryLimit {
		return Config{}, fmt.Errorf("history_limit must be at most %d", MaxHistoryLimit)
	}
	if cfg.Enabled && cfg.Topic == "" {
		return Config{}, fmt.Errorf("topic is required when clipboard sync is enabled")
	}
	if cfg.HistoryRetention != HistoryRetentionNone &&
		cfg.HistoryRetention != HistoryRetentionMetadata &&
		cfg.HistoryRetention != HistoryRetentionBody {
		return Config{}, fmt.Errorf("unsupported history_retention %q", cfg.HistoryRetention)
	}
	if cfg.TransferProvider == "" && cfg.TransferRef != "" {
		return Config{}, fmt.Errorf("transfer_provider is required when transfer_ref is set")
	}
	if cfg.TransferProvider != "" && cfg.TransferRef == "" {
		return Config{}, fmt.Errorf("transfer_ref is required when transfer_provider is set")
	}
	return cfg, nil
}
