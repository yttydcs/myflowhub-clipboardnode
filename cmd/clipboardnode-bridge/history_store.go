package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/bridge"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

type historyStore struct {
	path    string
	mu      sync.Mutex
	entries []bridge.HistoryEntry
}

func newHistoryStore(path string) (*historyStore, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("history path is required")
	}
	store := &historyStore{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return store, nil
	}
	var entries []bridge.HistoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("decode history: %w", err)
	}
	store.entries = normalizeLoadedHistory(entries)
	return store, nil
}

func (s *historyStore) Entries(cfg coreruntime.Config) []bridge.HistoryEntry {
	if s == nil || cfg.HistoryRetention != coreruntime.HistoryRetentionBody {
		return []bridge.HistoryEntry{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	limit := historyLimit(cfg)
	if len(s.entries) <= limit {
		return cloneHistoryEntries(s.entries)
	}
	return cloneHistoryEntries(s.entries[:limit])
}

func (s *historyStore) ApplySettings(cfg coreruntime.Config) (bool, error) {
	if s == nil {
		return false, nil
	}
	cfg, err := coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if cfg.HistoryRetention != coreruntime.HistoryRetentionBody {
		if len(s.entries) == 0 {
			return false, nil
		}
		s.entries = nil
		if err := s.saveLocked(); err != nil {
			return false, err
		}
		return true, nil
	}
	limit := historyLimit(cfg)
	if len(s.entries) <= limit {
		return false, nil
	}
	s.entries = cloneHistoryEntries(s.entries[:limit])
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *historyStore) AppendActivity(activity bridge.Activity, cfg coreruntime.Config) (bool, error) {
	if s == nil {
		return false, nil
	}
	cfg, err := coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return false, err
	}
	if cfg.HistoryRetention != coreruntime.HistoryRetentionBody {
		return false, nil
	}
	if activity.Text == "" || !historyKindCanPersist(activity.Kind) {
		return false, nil
	}
	entry := bridge.HistoryEntry{
		ID:          activity.ID,
		Kind:        activity.Kind,
		Text:        activity.Text,
		Topic:       activity.Topic,
		DeviceLabel: activity.DeviceLabel,
		ByteSize:    activity.ByteSize,
		HashPrefix:  activity.HashPrefix,
		TimestampMS: activity.TimestampMS,
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prependLocked(entry, historyLimit(cfg))
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *historyStore) Promote(entry bridge.HistoryEntry, cfg coreruntime.Config) (bool, error) {
	if s == nil {
		return false, nil
	}
	cfg, err := coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return false, err
	}
	if cfg.HistoryRetention != coreruntime.HistoryRetentionBody {
		return false, nil
	}
	if entry.Text == "" {
		return false, fmt.Errorf("history text is required")
	}
	now := time.Now()
	entry.ID = fmt.Sprintf("restore-%d", now.UnixMicro())
	entry.TimestampMS = now.UnixMilli()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.prependLocked(entry, historyLimit(cfg))
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *historyStore) Clear() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = nil
	return s.saveLocked()
}

func (s *historyStore) prependLocked(entry bridge.HistoryEntry, limit int) {
	entry = normalizeHistoryEntry(entry, time.Now())
	next := make([]bridge.HistoryEntry, 0, len(s.entries)+1)
	next = append(next, entry)
	for _, existing := range s.entries {
		if existing.Text == entry.Text {
			continue
		}
		next = append(next, existing)
		if len(next) >= limit {
			break
		}
	}
	s.entries = next
}

func (s *historyStore) saveLocked() error {
	entries := s.entries
	if entries == nil {
		entries = []bridge.HistoryEntry{}
	}
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("encode history: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".clipboardnode-history-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary history: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("restrict temporary history permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary history: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("flush temporary history: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary history: %w", err)
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace history: %w", err)
	}
	return nil
}

func normalizeLoadedHistory(entries []bridge.HistoryEntry) []bridge.HistoryEntry {
	out := make([]bridge.HistoryEntry, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	now := time.Now()
	for _, entry := range entries {
		if entry.Text == "" {
			continue
		}
		if _, ok := seen[entry.Text]; ok {
			continue
		}
		seen[entry.Text] = struct{}{}
		out = append(out, normalizeHistoryEntry(entry, now))
	}
	return out
}

func normalizeHistoryEntry(entry bridge.HistoryEntry, now time.Time) bridge.HistoryEntry {
	entry.ID = strings.TrimSpace(entry.ID)
	entry.Kind = strings.TrimSpace(entry.Kind)
	entry.Topic = strings.TrimSpace(entry.Topic)
	entry.DeviceLabel = strings.TrimSpace(entry.DeviceLabel)
	entry.HashPrefix = strings.TrimSpace(entry.HashPrefix)
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("history-%d", now.UnixMicro())
	}
	if entry.Kind == "" || !historyKindCanPersist(entry.Kind) {
		entry.Kind = "published"
	}
	if entry.ByteSize <= 0 {
		entry.ByteSize = len([]byte(entry.Text))
	}
	if entry.HashPrefix == "" {
		entry.HashPrefix = hashPrefixForText(entry.Text)
	}
	if entry.TimestampMS <= 0 {
		entry.TimestampMS = now.UnixMilli()
	}
	return entry
}

func historyKindCanPersist(kind string) bool {
	return kind == "published" || kind == "applied"
}

func historyLimit(cfg coreruntime.Config) int {
	if cfg.HistoryLimit <= 0 {
		return coreruntime.DefaultHistoryLimit
	}
	if cfg.HistoryLimit > coreruntime.MaxHistoryLimit {
		return coreruntime.MaxHistoryLimit
	}
	return cfg.HistoryLimit
}

func hashPrefixForText(text string) string {
	sum := sha256.Sum256([]byte(text))
	encoded := hex.EncodeToString(sum[:])
	if len(encoded) < 12 {
		return encoded
	}
	return encoded[:12]
}

func cloneHistoryEntries(entries []bridge.HistoryEntry) []bridge.HistoryEntry {
	if len(entries) == 0 {
		return []bridge.HistoryEntry{}
	}
	return append([]bridge.HistoryEntry(nil), entries...)
}
