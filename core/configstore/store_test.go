package configstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestStoreLoadMissingReturnsSafeDefaults(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "clipboardnode.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Enabled {
		t.Fatalf("default config should be disabled")
	}
	if cfg.MaxInlineBytes != coreruntime.DefaultMaxInlineBytes {
		t.Fatalf("max_inline_bytes = %d", cfg.MaxInlineBytes)
	}
	if cfg.ParentEndpoint != coreruntime.DefaultParentEndpoint {
		t.Fatalf("parent_endpoint = %q", cfg.ParentEndpoint)
	}
	if cfg.Topic != coreruntime.DefaultTopic ||
		len(cfg.Topics) != 1 ||
		cfg.Topics[0] != coreruntime.DefaultTopicRoute() {
		t.Fatalf("default topics = topic:%q topics:%+v", cfg.Topic, cfg.Topics)
	}
	if cfg.DeviceID != coreruntime.DefaultDeviceID || cfg.DisplayName != coreruntime.DefaultDeviceID {
		t.Fatalf("default identity fields = device_id:%q display_name:%q", cfg.DeviceID, cfg.DisplayName)
	}
}

func TestStoreSaveLoadDoesNotPersistClipboardText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboardnode.json")
	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := coreruntime.Config{
		Enabled:        true,
		ParentEndpoint: " 10.0.0.2:9000 ",
		Topic:          " clipboard/dev ",
		MaxInlineBytes: 1024,
		DeviceLabel:    " win-laptop ",
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "secret clipboard body") {
		t.Fatalf("config persisted clipboard text")
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if !loaded.Enabled ||
		loaded.ParentEndpoint != "10.0.0.2:9000" ||
		loaded.Topic != "clipboard/dev" ||
		len(loaded.Topics) != 1 ||
		loaded.Topics[0].Topic != "clipboard/dev" ||
		!loaded.Topics[0].SyncToLocal ||
		!loaded.Topics[0].SyncFromLocal ||
		loaded.DeviceID != "win-laptop" ||
		loaded.DisplayName != "win-laptop" ||
		loaded.DeviceLabel != "win-laptop" {
		t.Fatalf("loaded config = %+v", loaded)
	}
	if loaded.MaxInlineBytes != 1024 {
		t.Fatalf("max_inline_bytes = %d", loaded.MaxInlineBytes)
	}
}

func TestStoreSaveLoadKeepsDeviceIDAndDisplayNameSeparate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboardnode.json")
	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := coreruntime.Config{
		ParentEndpoint: "10.0.0.2:9000",
		Topic:          "clipboard/dev",
		MaxInlineBytes: 1024,
		DeviceID:       " device-a ",
		DisplayName:    " Laptop A ",
		DeviceLabel:    " legacy-label ",
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.DeviceID != "device-a" || loaded.DisplayName != "Laptop A" || loaded.DeviceLabel != "Laptop A" {
		t.Fatalf("loaded identity fields = %+v", loaded)
	}
}

func TestStoreLoadMigratesLegacyMetadataHistoryToBody(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboardnode.json")
	if err := os.WriteFile(
		path,
		[]byte(`{"history_retention":"metadata","max_inline_bytes":1024}`),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.HistoryRetention != coreruntime.HistoryRetentionBody {
		t.Fatalf("history_retention = %q", loaded.HistoryRetention)
	}
	if loaded.HistoryLimit != coreruntime.DefaultHistoryLimit {
		t.Fatalf("history_limit = %d", loaded.HistoryLimit)
	}
}

func TestStoreLoadPreservesExplicitMetadataHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboardnode.json")
	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := coreruntime.Config{
		ParentEndpoint:   "127.0.0.1:9000",
		MaxInlineBytes:   1024,
		HistoryRetention: coreruntime.HistoryRetentionMetadata,
		HistoryLimit:     32,
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.HistoryRetention != coreruntime.HistoryRetentionMetadata {
		t.Fatalf("history_retention = %q", loaded.HistoryRetention)
	}
	if loaded.HistoryLimit != 32 {
		t.Fatalf("history_limit = %d", loaded.HistoryLimit)
	}
}

func TestStoreRejectsExplicitEmptyTopics(t *testing.T) {
	store, err := New(filepath.Join(t.TempDir(), "clipboardnode.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = store.Save(coreruntime.Config{
		Enabled:        true,
		MaxInlineBytes: 1024,
		Topics:         []coreruntime.TopicRoute{},
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestStoreSaveLoadKeepsTopicRoutes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clipboardnode.json")
	store, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := coreruntime.Config{
		ParentEndpoint: "10.0.0.2:9000",
		MaxInlineBytes: 1024,
		Topics: []coreruntime.TopicRoute{
			{Topic: " clipboard/a ", SyncToLocal: true, SyncFromLocal: false},
			{Topic: "clipboard/b", SyncToLocal: false, SyncFromLocal: true},
		},
	}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Topic != "clipboard/a" {
		t.Fatalf("primary topic = %q", loaded.Topic)
	}
	if len(loaded.Topics) != 2 ||
		loaded.Topics[0].Topic != "clipboard/a" ||
		!loaded.Topics[0].SyncToLocal ||
		loaded.Topics[0].SyncFromLocal ||
		loaded.Topics[1].Topic != "clipboard/b" ||
		loaded.Topics[1].SyncToLocal ||
		!loaded.Topics[1].SyncFromLocal {
		t.Fatalf("loaded topics = %+v", loaded.Topics)
	}
}
