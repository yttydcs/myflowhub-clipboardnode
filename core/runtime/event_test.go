package runtime

import (
	"strings"
	"testing"
)

func TestNewClipboardTextEventV1_ValidatesAndHashes(t *testing.T) {
	evt, err := NewClipboardTextEventV1("hello", BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		OriginDevice:     "win-laptop",
		MaxInlineBytes:   64,
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
	raw, err := MarshalClipboardTextEventV1(evt, 64)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sha256") ||
		strings.Contains(string(raw), "size") ||
		strings.Contains(string(raw), "content_type") ||
		strings.Contains(string(raw), "encoding") ||
		strings.Contains(string(raw), "ts") {
		t.Fatalf("compact payload contains derived fields: %s", string(raw))
	}
}

func TestClipboardTextEventV1RejectsInvalidPayloads(t *testing.T) {
	valid, err := NewClipboardTextEventV1("hello", BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		MaxInlineBytes:   64,
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
		"zero origin node": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.OriginNode = 0
			return evt
		},
		"empty origin instance": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.OriginInstanceID = ""
			return evt
		},
		"empty text": func(evt ClipboardTextEventV1) ClipboardTextEventV1 {
			evt.Text = ""
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

func TestClipboardTransferManifestV1ValidatesWithoutBody(t *testing.T) {
	digest, err := InspectTextContent("oversize body")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := NewClipboardTransferManifestV1(digest, []TransferReference{
		{Provider: "stream", OpaqueID: "source-1"},
	}, BuildEventOptions{
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		OriginDevice:     "win-laptop",
		NewEventID:       func() (string, error) { return "transfer-1", nil },
	})
	if err != nil {
		t.Fatalf("NewClipboardTransferManifestV1 returned error: %v", err)
	}
	raw, err := MarshalClipboardTransferManifestV1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "oversize body") {
		t.Fatalf("manifest leaked clipboard body: %s", string(raw))
	}
	parsed, err := ParseClipboardTransferManifestV1(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.EventID != "transfer-1" || parsed.Size != len("oversize body") {
		t.Fatalf("unexpected manifest: %+v", parsed)
	}
}

func TestParseClipboardTextEventV1ComputesDigestFromCompactPayload(t *testing.T) {
	payload := []byte(`{"v":1,"id":"evt-1","from":12,"instance":"instance-a","text":"hello"}`)
	evt, err := ParseClipboardTextEventV1(payload, 64)
	if err != nil {
		t.Fatalf("ParseClipboardTextEventV1 returned error: %v", err)
	}
	if evt.Size != len("hello") || evt.SHA256 != HashText("hello") {
		t.Fatalf("derived digest = size %d hash %q", evt.Size, evt.SHA256)
	}
}

func TestParseClipboardTextEventV1RejectsOversizeCompactPayload(t *testing.T) {
	payload := []byte(`{"v":1,"id":"evt-1","from":12,"instance":"instance-a","text":"hello"}`)
	if _, err := ParseClipboardTextEventV1(payload, 4); err == nil {
		t.Fatalf("expected oversize error")
	}
}
