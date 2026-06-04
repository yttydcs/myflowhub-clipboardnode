package bridge

import (
	"encoding/json"
	"testing"
)

func TestCommandContractEncodesSetConfig(t *testing.T) {
	data, err := json.Marshal(Settings{
		Enabled:        true,
		ParentEndpoint: "10.0.0.2:9000",
		Topic:          "clipboard.text",
		Topics: []TopicRoute{
			{Topic: "clipboard.text", SyncToLocal: true, SyncFromLocal: true},
			{Topic: "clipboard/archive", SyncToLocal: false, SyncFromLocal: true},
		},
		DeviceID:         "device-a",
		DisplayName:      "Win Laptop",
		DeviceLabel:      "Win Laptop",
		MaxInlineBytes:   65536,
		AutoWatch:        true,
		AutoApply:        false,
		HistoryRetention: "body",
		HistoryLimit:     256,
	})
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := EncodeCommand(EngineCommand{
		ID:     "cmd-1",
		Action: ActionSetConfig,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := DecodeCommand(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "cmd-1" || got.Action != ActionSetConfig {
		t.Fatalf("unexpected command header: %+v", got)
	}
	var settings Settings
	if err := json.Unmarshal(got.Data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.ParentEndpoint != "10.0.0.2:9000" ||
		settings.Topic != "clipboard.text" ||
		len(settings.Topics) != 2 ||
		settings.Topics[0].Topic != "clipboard.text" ||
		!settings.Topics[0].SyncToLocal ||
		!settings.Topics[0].SyncFromLocal ||
		settings.Topics[1].Topic != "clipboard/archive" ||
		settings.Topics[1].SyncToLocal ||
		!settings.Topics[1].SyncFromLocal ||
		settings.DeviceID != "device-a" ||
		settings.DisplayName != "Win Laptop" ||
		settings.HistoryRetention != "body" ||
		settings.HistoryLimit != 256 ||
		!settings.AutoWatch ||
		settings.AutoApply {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestStatusEventOmitsClipboardBody(t *testing.T) {
	data, err := json.Marshal(Status{
		Connected:      true,
		LoggedIn:       true,
		ParentEndpoint: "10.0.0.2:9000",
		Enabled:        true,
		Topic:          "clipboard.text",
		Topics: []TopicRoute{
			{Topic: "clipboard.text", SyncToLocal: true, SyncFromLocal: true},
		},
		DeviceID:         "device-a",
		DisplayName:      "Desktop",
		DeviceLabel:      "Desktop",
		PendingEventID:   "evt-pending",
		PendingTopic:     "clipboard.pending",
		PendingSize:      12,
		PendingHash:      "123456abcdef",
		LastAction:       "local_published",
		LastEventID:      "evt-1",
		LastSize:         42,
		LastHashPrefix:   "abcdef123456",
		HistoryRetention: "body",
		HistoryLimit:     256,
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeEvent(EngineEvent{Name: EventStatusChanged, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" {
		t.Fatal("empty event")
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["text"]; ok {
		t.Fatal("status event leaked clipboard text field")
	}
	var event EngineEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		t.Fatal(err)
	}
	var status map[string]json.RawMessage
	if err := json.Unmarshal(event.Data, &status); err != nil {
		t.Fatal(err)
	}
	if _, ok := status["text"]; ok {
		t.Fatal("status data leaked clipboard text field")
	}
	var decoded Status
	if err := json.Unmarshal(event.Data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PendingTopic != "clipboard.pending" || decoded.PendingEventID != "evt-pending" {
		t.Fatalf("unexpected pending status fields: %+v", decoded)
	}
}

func TestActivityEventCanCarryExplicitHistoryText(t *testing.T) {
	data, err := json.Marshal(Activity{
		ID:          "evt-1",
		Kind:        "published",
		Title:       "local_published",
		Detail:      "TopicBus: clipboard.text",
		Topic:       "clipboard.text",
		ByteSize:    5,
		HashPrefix:  "abcdef123456",
		TimestampMS: 1700000000000,
		Text:        "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeEvent(EngineEvent{Name: EventActivityUpdated, Data: data, OK: true})
	if err != nil {
		t.Fatal(err)
	}
	var event EngineEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		t.Fatal(err)
	}
	var activity Activity
	if err := json.Unmarshal(event.Data, &activity); err != nil {
		t.Fatal(err)
	}
	if activity.Text != "hello" || activity.ByteSize != 5 || activity.Topic != "clipboard.text" {
		t.Fatalf("unexpected activity: %+v", activity)
	}
}

func TestHistoryEventCarriesDedicatedBodyList(t *testing.T) {
	data, err := json.Marshal([]HistoryEntry{
		{
			ID:          "evt-1",
			Kind:        "published",
			Text:        "history body",
			Topic:       "clipboard.text",
			DeviceLabel: "Desktop",
			ByteSize:    12,
			HashPrefix:  "abcdef123456",
			TimestampMS: 1700000000000,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeEvent(EngineEvent{Name: EventHistoryUpdated, Data: data, OK: true})
	if err != nil {
		t.Fatal(err)
	}
	var event EngineEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		t.Fatal(err)
	}
	if event.Name != EventHistoryUpdated {
		t.Fatalf("event name = %q", event.Name)
	}
	var entries []HistoryEntry
	if err := json.Unmarshal(event.Data, &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Text != "history body" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestRestoreHistoryCommandEncodesHistoryEntry(t *testing.T) {
	data, err := json.Marshal(HistoryEntry{
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
	encoded, err := EncodeCommand(EngineCommand{
		ID:     "cmd-restore",
		Action: ActionRestoreHistory,
		Data:   data,
	})
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeCommand(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Action != ActionRestoreHistory {
		t.Fatalf("action = %q", decoded.Action)
	}
	var entry HistoryEntry
	if err := json.Unmarshal(decoded.Data, &entry); err != nil {
		t.Fatal(err)
	}
	if entry.Text != "restore body" || entry.Topic != "clipboard.text" {
		t.Fatalf("entry = %+v", entry)
	}
}

func TestErrorEventEncodesExplicitFalseOK(t *testing.T) {
	encoded, err := EncodeEvent(EngineEvent{Name: EventError, OK: false, Error: "bad command"})
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatal(err)
	}
	okValue, exists := raw["ok"]
	if !exists {
		t.Fatal("error event omitted ok field")
	}
	var ok bool
	if err := json.Unmarshal(okValue, &ok); err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("error event encoded ok=true")
	}
}

func TestTransferEventOmitsClipboardBody(t *testing.T) {
	data, err := json.Marshal(Transfer{
		ID:         "transfer-1",
		State:      "manifest_published",
		ByteSize:   128000,
		HashPrefix: "abcdef123456",
		Detail:     "clipboard.transfer.v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeEvent(EngineEvent{Name: EventTransferUpdate, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) == "" {
		t.Fatal("empty event")
	}
	if string(encoded) == "" {
		t.Fatal("empty event")
	}
	var transfer Transfer
	var event EngineEvent
	if err := json.Unmarshal(encoded, &event); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(event.Data, &transfer); err != nil {
		t.Fatal(err)
	}
	if transfer.Detail != "clipboard.transfer.v1" || transfer.ByteSize != 128000 {
		t.Fatalf("unexpected transfer event: %+v", transfer)
	}
}
