package runtime

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewClipboardTextEventV1_ValidatesAndHashes(t *testing.T) {
	evt, err := NewClipboardTextEventV1("hello", BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		OriginDevice:     "win-laptop",
		MaxInlineBytes:   64,
		Now:              func() time.Time { return time.UnixMilli(1760000000000) },
		NewEventID:       func() (string, error) { return "evt-1", nil },
	})
	if err != nil {
		t.Fatalf("NewClipboardTextEventV1 returned error: %v", err)
	}
	if evt.Version != EventVersionV1 {
		t.Fatalf("version = %d", evt.Version)
	}
	if evt.EventID != "evt-1" {
		t.Fatalf("event id = %q", evt.EventID)
	}
	if evt.Size != 5 {
		t.Fatalf("size = %d", evt.Size)
	}
	if evt.SHA256 != HashText("hello") {
		t.Fatalf("sha256 = %q", evt.SHA256)
	}
	if evt.TS != 1760000000000 {
		t.Fatalf("ts = %d", evt.TS)
	}
}

func TestClipboardTextEventV1RejectsInvalidPayloads(t *testing.T) {
	valid, err := NewClipboardTextEventV1("hello", BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		MaxInlineBytes:   64,
		Now:              func() time.Time { return time.Unix(1, 0) },
		NewEventID:       func() (string, error) { return "evt-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]func(ClipboardTextEventV1) ClipboardTextEventV1{
		"unsupported version": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.Version = 2
			return evt
		},
		"empty event id": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.EventID = ""
			return evt
		},
		"unsupported content type": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.ContentType = "text/html"
			return evt
		},
		"unsupported encoding": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.Encoding = "utf-16"
			return evt
		},
		"size mismatch": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.Size = 999
			return evt
		},
		"hash mismatch": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.SHA256 = strings.Repeat("0", 64)
			return evt
		},
		"empty text": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.Text = ""
			evt.Size = 0
			evt.SHA256 = HashText("")
			return evt
		},
	}

	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			if err := mutate(valid).Validate(64); err == nil {
				t.Fatalf("expected validation error")
			}
		})
	}
}

func TestInspectTextRejectsOversize(t *testing.T) {
	_, err := InspectText("hello", 4)
	if err == nil {
		t.Fatalf("expected oversize error")
	}
}

func TestParseClipboardTextEventV1RejectsHashMismatch(t *testing.T) {
	evt, err := NewClipboardTextEventV1("hello", BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		MaxInlineBytes:   64,
		Now:              func() time.Time { return time.Unix(1, 0) },
		NewEventID:       func() (string, error) { return "evt-1", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	evt.Text = "tampered"
	payload, err := json.Marshal(evt)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseClipboardTextEventV1(payload, 64); err == nil {
		t.Fatalf("expected hash mismatch error")
	}
}
