package configstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

type Store struct {
	path string
}

func New(path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is required")
	}
	return &Store{path: path}, nil
}

func (s *Store) Load() (coreruntime.Config, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return coreruntime.DefaultConfig(), nil
	}
	if err != nil {
		return coreruntime.Config{}, fmt.Errorf("read config: %w", err)
	}
	var cfg coreruntime.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return coreruntime.Config{}, fmt.Errorf("decode config: %w", err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return coreruntime.Config{}, fmt.Errorf("decode config fields: %w", err)
	}
	cfg = migrateLegacyHistoryDefaults(cfg, fields)
	cfg, err = coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return coreruntime.Config{}, fmt.Errorf("validate config: %w", err)
	}
	return cfg, nil
}

func migrateLegacyHistoryDefaults(cfg coreruntime.Config, fields map[string]json.RawMessage) coreruntime.Config {
	if cfg.HistoryRetention != coreruntime.HistoryRetentionMetadata {
		return cfg
	}
	if raw, ok := fields["history_limit"]; ok && string(raw) != "null" {
		return cfg
	}
	cfg.HistoryRetention = coreruntime.HistoryRetentionBody
	if cfg.HistoryLimit == 0 {
		cfg.HistoryLimit = coreruntime.DefaultHistoryLimit
	}
	return cfg
}

func (s *Store) Save(cfg coreruntime.Config) error {
	cfg, err := coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return fmt.Errorf("validate config: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".clipboardnode-config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary config: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict temporary config permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary config: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("flush temporary config: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary config: %w", err)
	}
	if err := replaceFile(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace config: %w", err)
	}
	return nil
}
