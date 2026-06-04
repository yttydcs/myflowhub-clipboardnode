package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yttydcs/myflowhub-clipboardnode/bridge"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestHistoryStoreLoadMissingReturnsEmpty(t *testing.T) {
	store, err := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	if err != nil {
		t.Fatal(err)
	}
	entries := store.Entries(historyConfig(10, coreruntime.HistoryRetentionBody))
	if len(entries) != 0 {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestHistoryStoreAppendPersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	store, err := newHistoryStore(path)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := store.AppendActivity(bridge.Activity{
		ID:          "evt-1",
		Kind:        "published",
		Text:        "persisted clipboard body",
		Topic:       "clipboard.text",
		DeviceLabel: "device-a",
		ByteSize:    24,
		HashPrefix:  "abcdef123456",
		TimestampMS: 1700000000000,
	}, historyConfig(10, coreruntime.HistoryRetentionBody))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed history")
	}

	reloaded, err := newHistoryStore(path)
	if err != nil {
		t.Fatal(err)
	}
	entries := reloaded.Entries(historyConfig(10, coreruntime.HistoryRetentionBody))
	if len(entries) != 1 || entries[0].Text != "persisted clipboard body" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestHistoryStoreDedupesAndTrims(t *testing.T) {
	store, err := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	if err != nil {
		t.Fatal(err)
	}
	cfg := historyConfig(2, coreruntime.HistoryRetentionBody)
	for _, text := range []string{"alpha", "beta", "alpha", "gamma"} {
		if _, err := store.AppendActivity(bridge.Activity{
			ID:   "evt-" + text,
			Kind: "published",
			Text: text,
		}, cfg); err != nil {
			t.Fatal(err)
		}
	}

	entries := store.Entries(cfg)
	if len(entries) != 2 || entries[0].Text != "gamma" || entries[1].Text != "alpha" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestHistoryStoreApplyMetadataRetentionClearsPersistedHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	store, err := newHistoryStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendActivity(bridge.Activity{
		ID:   "evt-1",
		Kind: "applied",
		Text: "clear me",
	}, historyConfig(10, coreruntime.HistoryRetentionBody)); err != nil {
		t.Fatal(err)
	}

	changed, err := store.ApplySettings(historyConfig(10, coreruntime.HistoryRetentionMetadata))
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed history")
	}
	if entries := store.Entries(historyConfig(10, coreruntime.HistoryRetentionBody)); len(entries) != 0 {
		t.Fatalf("entries = %+v", entries)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]\n" {
		t.Fatalf("history file = %q", string(data))
	}
}

func TestHistoryStorePromotePersistsTopEntry(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history.json")
	store, err := newHistoryStore(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg := historyConfig(10, coreruntime.HistoryRetentionBody)
	if _, err := store.AppendActivity(bridge.Activity{
		ID:   "evt-alpha",
		Kind: "published",
		Text: "alpha",
	}, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AppendActivity(bridge.Activity{
		ID:   "evt-beta",
		Kind: "published",
		Text: "beta",
	}, cfg); err != nil {
		t.Fatal(err)
	}
	changed, err := store.Promote(bridge.HistoryEntry{
		ID:   "evt-alpha",
		Kind: "published",
		Text: "alpha",
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected changed history")
	}

	reloaded, err := newHistoryStore(path)
	if err != nil {
		t.Fatal(err)
	}
	entries := reloaded.Entries(cfg)
	if len(entries) != 2 || entries[0].Text != "alpha" || entries[1].Text != "beta" {
		t.Fatalf("entries = %+v", entries)
	}
	if entries[0].ID == "evt-alpha" || entries[0].ID == "" {
		t.Fatalf("promoted id = %q", entries[0].ID)
	}
}

func historyConfig(limit int, retention string) coreruntime.Config {
	cfg := coreruntime.DefaultConfig()
	cfg.HistoryLimit = limit
	cfg.HistoryRetention = retention
	return cfg
}
