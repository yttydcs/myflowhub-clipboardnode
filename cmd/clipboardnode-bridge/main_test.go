package main

import (
	"testing"

	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestActivityFromDecisionIncludesTextOnlyForBodyHistory(t *testing.T) {
	decision := coreruntime.Decision{
		Action:     coreruntime.ActionLocalPublished,
		EventID:    "evt-1",
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
}
