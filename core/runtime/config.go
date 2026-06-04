package runtime

import (
	"fmt"
	"strings"
)

const (
	DefaultParentEndpoint    = "127.0.0.1:9000"
	DefaultTopic             = "clipboard.text"
	DefaultMaxInlineBytes    = 65536
	DefaultDeviceID          = "local-device"
	DefaultHistoryLimit      = 256
	MaxTopicRoutes           = 32
	MaxHistoryLimit          = 5000
	HistoryRetentionNone     = "none"
	HistoryRetentionMetadata = "metadata"
	HistoryRetentionBody     = "body"
	defaultRecentLimit       = 128
	defaultSuppressLimit     = 32
)

type TopicRoute struct {
	Topic         string `json:"topic"`
	SyncToLocal   bool   `json:"sync_to_local"`
	SyncFromLocal bool   `json:"sync_from_local"`
}

type Config struct {
	Enabled          bool         `json:"enabled"`
	ParentEndpoint   string       `json:"parent_endpoint"`
	Topic            string       `json:"topic"`
	Topics           []TopicRoute `json:"topics,omitempty"`
	MaxInlineBytes   int          `json:"max_inline_bytes"`
	DeviceID         string       `json:"device_id,omitempty"`
	DisplayName      string       `json:"display_name,omitempty"`
	DeviceLabel      string       `json:"device_label,omitempty"`
	AutoWatch        bool         `json:"auto_watch"`
	AutoApply        bool         `json:"auto_apply"`
	HistoryRetention string       `json:"history_retention,omitempty"`
	HistoryLimit     int          `json:"history_limit,omitempty"`
	TransferProvider string       `json:"transfer_provider,omitempty"`
	TransferRef      string       `json:"transfer_ref,omitempty"`
}

func DefaultConfig() Config {
	return Config{
		ParentEndpoint:   DefaultParentEndpoint,
		Enabled:          false,
		Topic:            DefaultTopic,
		Topics:           []TopicRoute{DefaultTopicRoute()},
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

func DefaultTopicRoute() TopicRoute {
	return TopicRoute{
		Topic:         DefaultTopic,
		SyncToLocal:   true,
		SyncFromLocal: true,
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
	routes, err := NormalizeTopicRoutes(cfg.Topic, cfg.Topics)
	if err != nil {
		return Config{}, err
	}
	cfg.Topics = routes
	cfg.Topic = routes[0].Topic
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
	if cfg.Enabled && len(cfg.Topics) == 0 {
		return Config{}, fmt.Errorf("at least one topic is required when clipboard sync is enabled")
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

func NormalizeTopicRoutes(legacyTopic string, routes []TopicRoute) ([]TopicRoute, error) {
	legacyTopic = strings.TrimSpace(legacyTopic)
	if routes == nil {
		if legacyTopic == "" {
			legacyTopic = DefaultTopic
		}
		return []TopicRoute{{
			Topic:         legacyTopic,
			SyncToLocal:   true,
			SyncFromLocal: true,
		}}, nil
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("at least one topic route is required")
	}
	if len(routes) > MaxTopicRoutes {
		return nil, fmt.Errorf("topics must contain at most %d entries", MaxTopicRoutes)
	}
	out := make([]TopicRoute, 0, len(routes))
	seen := make(map[string]struct{}, len(routes))
	for i, route := range routes {
		route.Topic = strings.TrimSpace(route.Topic)
		if route.Topic == "" {
			return nil, fmt.Errorf("topics[%d].topic is required", i)
		}
		if _, ok := seen[route.Topic]; ok {
			return nil, fmt.Errorf("duplicate topic %q", route.Topic)
		}
		seen[route.Topic] = struct{}{}
		out = append(out, route)
	}
	return out, nil
}

func CloneTopicRoutes(routes []TopicRoute) []TopicRoute {
	if len(routes) == 0 {
		return nil
	}
	return append([]TopicRoute(nil), routes...)
}
