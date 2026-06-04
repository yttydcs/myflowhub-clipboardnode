package nodemobile

import (
	"context"
	"encoding/json"
	"testing"

	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestManualClipboardTracksOnlyRemoteAppliedText(t *testing.T) {
	manual := &manualClipboard{}

	manual.SetLocalText("local text")
	if got := manual.TakeLastAppliedText(); got != "" {
		t.Fatalf("local text marked applied = %q", got)
	}

	if err := manual.WriteText(context.Background(), "remote text"); err != nil {
		t.Fatalf("write text: %v", err)
	}
	if got := manual.TakeLastAppliedText(); got != "remote text" {
		t.Fatalf("last applied text = %q", got)
	}
	if got := manual.TakeLastAppliedText(); got != "" {
		t.Fatalf("last applied text was not cleared = %q", got)
	}

	if got, err := manual.ReadText(context.Background()); err != nil || got != "remote text" {
		t.Fatalf("read text = %q, %v", got, err)
	}
}

func TestMarshalDecisionOnlyIncludesBodyForLocalAndApplied(t *testing.T) {
	local := decodeDecisionForTest(t, marshalDecision(coreruntime.Decision{
		Action: coreruntime.ActionLocalPublished,
		Text:   "local body",
	}))
	if local["Text"] != "local body" {
		t.Fatalf("local published text = %#v", local["Text"])
	}

	applied := decodeDecisionForTest(t, marshalDecision(coreruntime.Decision{
		Action: coreruntime.ActionRemoteApplied,
		Text:   "remote body",
	}))
	if applied["Text"] != "remote body" {
		t.Fatalf("remote applied text = %#v", applied["Text"])
	}

	pending := decodeDecisionForTest(t, marshalDecision(coreruntime.Decision{
		Action: coreruntime.ActionRemotePending,
		Text:   "pending body",
	}))
	if _, ok := pending["Text"]; ok {
		t.Fatalf("pending decision leaked text: %#v", pending)
	}
}

func decodeDecisionForTest(t *testing.T, raw string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("decode decision: %v", err)
	}
	return out
}
