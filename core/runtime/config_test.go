package runtime

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestNormalizeConfigDefaultsBodyHistory(t *testing.T) {
	cfg, err := NormalizeConfig(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Topic != DefaultTopic {
		t.Fatalf("topic = %q", cfg.Topic)
	}
	if len(cfg.Topics) != 1 || cfg.Topics[0] != DefaultTopicRoute() {
		t.Fatalf("topics = %+v", cfg.Topics)
	}
	if cfg.HistoryRetention != HistoryRetentionBody {
		t.Fatalf("history retention = %q", cfg.HistoryRetention)
	}
	if cfg.HistoryLimit != DefaultHistoryLimit {
		t.Fatalf("history limit = %d", cfg.HistoryLimit)
	}
}

func TestNormalizeConfigValidatesHistoryLimit(t *testing.T) {
	tests := map[string]Config{
		"negative":  {HistoryLimit: -1},
		"too large": {HistoryLimit: MaxHistoryLimit + 1},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NormalizeConfig(cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "history_limit") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestNormalizeConfigTopicRoutes(t *testing.T) {
	cfg, err := NormalizeConfig(Config{
		Topic: " clipboard/legacy ",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Topic != "clipboard/legacy" {
		t.Fatalf("legacy topic = %q", cfg.Topic)
	}
	if len(cfg.Topics) != 1 || cfg.Topics[0].Topic != "clipboard/legacy" || !cfg.Topics[0].SyncToLocal || !cfg.Topics[0].SyncFromLocal {
		t.Fatalf("legacy topics = %+v", cfg.Topics)
	}

	cfg, err = NormalizeConfig(Config{
		Topics: []TopicRoute{
			{Topic: " clipboard/a ", SyncToLocal: true},
			{Topic: "clipboard/b", SyncFromLocal: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Topic != "clipboard/a" {
		t.Fatalf("primary topic = %q", cfg.Topic)
	}
	if len(cfg.Topics) != 2 ||
		cfg.Topics[0].Topic != "clipboard/a" ||
		!cfg.Topics[0].SyncToLocal ||
		cfg.Topics[0].SyncFromLocal ||
		cfg.Topics[1].Topic != "clipboard/b" ||
		cfg.Topics[1].SyncToLocal ||
		!cfg.Topics[1].SyncFromLocal {
		t.Fatalf("topics = %+v", cfg.Topics)
	}
}

func TestNormalizeConfigRejectsInvalidTopicRoutes(t *testing.T) {
	tests := map[string]Config{
		"empty list": {
			Topics: []TopicRoute{},
		},
		"empty route": {
			Topics: []TopicRoute{{Topic: " "}},
		},
		"duplicate route": {
			Topics: []TopicRoute{
				{Topic: "clipboard/a"},
				{Topic: " clipboard/a "},
			},
		},
	}
	for name, cfg := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NormalizeConfig(cfg)
			if err == nil {
				t.Fatal("expected validation error")
			}
			if !strings.Contains(err.Error(), "topic") {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestNormalizeConfigAcceptsHistoryRetentionModes(t *testing.T) {
	for _, retention := range []string{
		HistoryRetentionNone,
		HistoryRetentionMetadata,
		HistoryRetentionBody,
	} {
		t.Run(retention, func(t *testing.T) {
			cfg, err := NormalizeConfig(Config{HistoryRetention: retention, HistoryLimit: 16})
			if err != nil {
				t.Fatal(err)
			}
			if cfg.HistoryRetention != retention || cfg.HistoryLimit != 16 {
				t.Fatalf("config = %+v", cfg)
			}
		})
	}
}

func TestDecisionJSONOmitsText(t *testing.T) {
	raw, err := json.Marshal(Decision{
		Action:     ActionLocalPublished,
		EventID:    "evt-1",
		Size:       5,
		HashPrefix: "abcdef123456",
		Text:       "secret body",
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "secret body") || strings.Contains(string(raw), "Text") {
		t.Fatalf("decision json leaked text: %s", string(raw))
	}
}
