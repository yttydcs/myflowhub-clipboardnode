package bridge

import (
	"encoding/json"
	"testing"
)

func TestCommandContractEncodesSetConfig(t *testing.T) {
	data, err := json.Marshal(Settings{
		Enabled:          true,
		ParentEndpoint:   "10.0.0.2:9000",
		Topic:            "clipboard/shared",
		DeviceLabel:      "win-laptop",
		MaxInlineBytes:   65536,
		AutoWatch:        true,
		AutoApply:        false,
		HistoryRetention: "metadata",
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
		settings.Topic != "clipboard/shared" ||
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
		Topic:          "clipboard/shared",
		DeviceLabel:    "desktop",
		LastAction:     "local_published",
		LastEventID:    "evt-1",
		LastSize:       42,
		LastHashPrefix: "abcdef123456",
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
}
