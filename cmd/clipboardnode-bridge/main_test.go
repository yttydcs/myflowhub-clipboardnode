package main

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/yttydcs/myflowhub-clipboardnode/bridge"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestActivityFromDecisionIncludesTextOnlyForBodyHistory(t *testing.T) {
	decision := coreruntime.Decision{
		Action:     coreruntime.ActionLocalPublished,
		EventID:    "evt-1",
		Topic:      "clipboard.text",
		Size:       5,
		HashPrefix: "abcdef123456",
		Text:       "hello",
	}

	body := activityFromDecision(decision, coreruntime.Config{
		HistoryRetention: coreruntime.HistoryRetentionBody,
	}, 1700000000000)
	if body.Text != "hello" {
		t.Fatalf("body activity text = %q", body.Text)
	}
	if body.Topic != "clipboard.text" || body.Detail != "TopicBus: clipboard.text" {
		t.Fatalf("body activity topic fields = topic:%q detail:%q", body.Topic, body.Detail)
	}

	metadata := activityFromDecision(decision, coreruntime.Config{
		HistoryRetention: coreruntime.HistoryRetentionMetadata,
	}, 1700000000000)
	if metadata.Text != "" {
		t.Fatalf("metadata activity leaked text = %q", metadata.Text)
	}

	none := activityFromDecision(decision, coreruntime.Config{
		HistoryRetention: coreruntime.HistoryRetentionNone,
	}, 1700000000000)
	if none.Text != "" {
		t.Fatalf("none activity leaked text = %q", none.Text)
	}

	pending := activityFromDecision(coreruntime.Decision{
		Action:     coreruntime.ActionRemotePending,
		EventID:    "evt-pending",
		Topic:      "clipboard.text",
		Size:       5,
		HashPrefix: "abcdef123456",
		Text:       "pending body",
	}, coreruntime.Config{
		HistoryRetention: coreruntime.HistoryRetentionBody,
	}, 1700000000000)
	if pending.Text != "" {
		t.Fatalf("pending activity leaked text = %q", pending.Text)
	}
}

func TestHandleRestoreHistoryPersistsAndEmitsHistoryUpdate(t *testing.T) {
	store, err := newHistoryStore(filepath.Join(t.TempDir(), "history.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	host := &stdioHost{
		history: store,
		cfg:     coreruntime.DefaultConfig(),
		out:     &out,
	}
	data, err := json.Marshal(bridge.HistoryEntry{
		ID:          "evt-1",
		Kind:        "published",
		Text:        "restore body",
		Topic:       "clipboard.text",
		DeviceLabel: "Desktop",
		ByteSize:    12,
		HashPrefix:  "abcdef123456",
		TimestampMS: 1700000000000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.handle(context.Background(), bridge.EngineCommand{
		ID:     "cmd-restore",
		Action: bridge.ActionRestoreHistory,
		Data:   data,
	}); err != nil {
		t.Fatal(err)
	}

	var event bridge.EngineEvent
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &event); err != nil {
		t.Fatalf("decode event: %v; raw=%q", err, out.String())
	}
	if event.ID != "cmd-restore" || event.Name != bridge.EventHistoryUpdated || !event.OK {
		t.Fatalf("event = %+v", event)
	}
	var entries []bridge.HistoryEntry
	if err := json.Unmarshal(event.Data, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "restore body" || entries[0].ID == "evt-1" {
		t.Fatalf("entries = %+v", entries)
	}
}
