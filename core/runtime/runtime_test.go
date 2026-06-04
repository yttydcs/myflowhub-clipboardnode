package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

type fakeTopicBus struct {
	mu             sync.Mutex
	subscribed     []string
	unsubscribed   []string
	published      []TopicBusMessage
	subscribeErr   error
	subscribeErrOn string
	publishErr     error
	unsubscribeErr error
}

func (f *fakeTopicBus) Subscribe(_ context.Context, topic string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subscribeErr != nil && (f.subscribeErrOn == "" || f.subscribeErrOn == topic) {
		return f.subscribeErr
	}
	f.subscribed = append(f.subscribed, topic)
	return nil
}

func (f *fakeTopicBus) Unsubscribe(_ context.Context, topic string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.unsubscribeErr != nil {
		return f.unsubscribeErr
	}
	f.unsubscribed = append(f.unsubscribed, topic)
	return nil
}

func (f *fakeTopicBus) Publish(_ context.Context, topic string, name string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.publishErr != nil {
		return f.publishErr
	}
	copied := append([]byte(nil), payload...)
	f.published = append(f.published, TopicBusMessage{Topic: topic, Name: name, Payload: copied})
	return nil
}

func (f *fakeTopicBus) publishCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

type fakeClipboard struct {
	mu       sync.Mutex
	text     string
	writes   []string
	events   chan clipboard.TextEvent
	watches  int
	watchErr error
	writeErr error
	closed   bool
}

func newFakeClipboard() *fakeClipboard {
	return &fakeClipboard{events: make(chan clipboard.TextEvent, 8)}
}

func (f *fakeClipboard) ReadText(_ context.Context) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.text, nil
}

func (f *fakeClipboard) WriteText(_ context.Context, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.writeErr != nil {
		return f.writeErr
	}
	f.text = text
	f.writes = append(f.writes, text)
	return nil
}

func (f *fakeClipboard) WatchText(ctx context.Context) (<-chan clipboard.TextEvent, error) {
	if f.watchErr != nil {
		return nil, f.watchErr
	}
	f.mu.Lock()
	f.watches++
	f.mu.Unlock()
	out := make(chan clipboard.TextEvent)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt := <-f.events:
				select {
				case <-ctx.Done():
					return
				case out <- evt:
				}
			}
		}
	}()
	return out, nil
}

func (f *fakeClipboard) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

func TestRuntimeHandleLocalTextPublishesValidEvent(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)

	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "hello"})
	if err != nil {
		t.Fatalf("HandleLocalText returned error: %v", err)
	}
	if decision.Action != ActionLocalPublished {
		t.Fatalf("action = %s", decision.Action)
	}
	if decision.Text != "hello" {
		t.Fatalf("decision text = %q", decision.Text)
	}
	if bus.publishCount() != 1 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
	got := bus.published[0]
	if got.Topic != "clipboard/dev" || got.Name != ClipboardTextEventName {
		t.Fatalf("unexpected topicbus message: %+v", got)
	}
	if strings.Contains(string(got.Payload), "sha256") ||
		strings.Contains(string(got.Payload), "size") ||
		strings.Contains(string(got.Payload), "content_type") ||
		strings.Contains(string(got.Payload), "encoding") {
		t.Fatalf("published text payload is not compact: %s", string(got.Payload))
	}
	evt, err := ParseClipboardTextEventV1(got.Payload, 64)
	if err != nil {
		t.Fatal(err)
	}
	if evt.Text != "hello" {
		t.Fatalf("event text = %q", evt.Text)
	}
	if evt.Size != len("hello") || evt.SHA256 != HashText("hello") {
		t.Fatalf("derived digest = size %d hash %q", evt.Size, evt.SHA256)
	}
}

func TestRuntimeLocalPublishFansOutToSyncFromLocalTopics(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			MaxInlineBytes: 64,
			AutoApply:      true,
			Topics: []TopicRoute{
				{Topic: "clipboard/a", SyncToLocal: true, SyncFromLocal: true},
				{Topic: "clipboard/b", SyncToLocal: true, SyncFromLocal: false},
				{Topic: "clipboard/c", SyncToLocal: false, SyncFromLocal: true},
			},
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
		NewEventID: func() (string, error) { return "evt-local", nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "hello"})
	if err != nil {
		t.Fatalf("HandleLocalText returned error: %v", err)
	}
	if decision.Action != ActionLocalPublished {
		t.Fatalf("action = %s", decision.Action)
	}
	if bus.publishCount() != 2 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
	gotTopics := []string{bus.published[0].Topic, bus.published[1].Topic}
	if strings.Join(gotTopics, ",") != "clipboard/a,clipboard/c" {
		t.Fatalf("published topics = %v", gotTopics)
	}
}

func TestRuntimeLocalPublishIgnoredWhenNoSyncFromLocalTopic(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			MaxInlineBytes: 64,
			Topics: []TopicRoute{
				{Topic: "clipboard/read-only", SyncToLocal: true, SyncFromLocal: false},
			},
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "hello"})
	if err != nil {
		t.Fatalf("HandleLocalText returned error: %v", err)
	}
	if decision.Action != ActionIgnoredLocalPolicy {
		t.Fatalf("action = %s", decision.Action)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
}

func TestRuntimeDisabledDoesNotPublishOrApply(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config:     DefaultConfig(),
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "hello"})
	if err != nil {
		t.Fatalf("disabled local returned error: %v", err)
	}
	if decision.Action != ActionDisabled {
		t.Fatalf("action = %s", decision.Action)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}

	msg := buildRemoteMessage(t, "evt-remote", "remote text")
	decision, err = rt.HandleTopicBusMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("disabled remote returned error: %v", err)
	}
	if decision.Action != ActionDisabled {
		t.Fatalf("remote action = %s", decision.Action)
	}
	if len(clip.writes) != 0 {
		t.Fatalf("writes = %v", clip.writes)
	}
}

func TestRuntimeRemoteApplyDuplicateAndLoopSuppression(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)

	msg := buildRemoteMessage(t, "evt-remote", "remote text")
	decision, err := rt.HandleTopicBusMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("remote apply returned error: %v", err)
	}
	if decision.Action != ActionRemoteApplied {
		t.Fatalf("action = %s", decision.Action)
	}
	if decision.Text != "remote text" {
		t.Fatalf("decision text = %q", decision.Text)
	}
	if len(clip.writes) != 1 || clip.writes[0] != "remote text" {
		t.Fatalf("writes = %v", clip.writes)
	}

	decision, err = rt.HandleTopicBusMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("duplicate returned error: %v", err)
	}
	if decision.Action != ActionIgnoredDuplicate {
		t.Fatalf("duplicate action = %s", decision.Action)
	}
	if decision.Text != "" {
		t.Fatalf("duplicate decision leaked text = %q", decision.Text)
	}

	decision, err = rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "remote text"})
	if err != nil {
		t.Fatalf("loop local returned error: %v", err)
	}
	if decision.Action != ActionIgnoredLoop {
		t.Fatalf("loop action = %s", decision.Action)
	}
	if decision.Text != "" {
		t.Fatalf("loop decision leaked text = %q", decision.Text)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
}

func TestRuntimeRemoteApplyHonorsTopicSyncToLocalPolicy(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			MaxInlineBytes: 64,
			AutoApply:      true,
			Topics: []TopicRoute{
				{Topic: "clipboard/read", SyncToLocal: true, SyncFromLocal: false},
				{Topic: "clipboard/write", SyncToLocal: false, SyncFromLocal: true},
			},
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}

	decision, err := rt.HandleTopicBusMessage(context.Background(), buildRemoteMessageOnTopic(t, "clipboard/write", "evt-write", "write-only text"))
	if err != nil {
		t.Fatalf("write-only topic returned error: %v", err)
	}
	if decision.Action != ActionIgnoredTopicPolicy || decision.Topic != "clipboard/write" {
		t.Fatalf("write-only decision = %+v", decision)
	}
	if len(clip.writes) != 0 {
		t.Fatalf("writes = %v", clip.writes)
	}

	decision, err = rt.HandleTopicBusMessage(context.Background(), buildRemoteMessageOnTopic(t, "clipboard/unknown", "evt-unknown", "unknown text"))
	if err != nil {
		t.Fatalf("unknown topic returned error: %v", err)
	}
	if decision.Action != ActionIgnoredTopic || decision.Topic != "clipboard/unknown" {
		t.Fatalf("unknown decision = %+v", decision)
	}

	decision, err = rt.HandleTopicBusMessage(context.Background(), buildRemoteMessageOnTopic(t, "clipboard/read", "evt-read", "read text"))
	if err != nil {
		t.Fatalf("read topic returned error: %v", err)
	}
	if decision.Action != ActionRemoteApplied || decision.Topic != "clipboard/read" {
		t.Fatalf("read decision = %+v", decision)
	}
	if len(clip.writes) != 1 || clip.writes[0] != "read text" {
		t.Fatalf("writes after read topic = %v", clip.writes)
	}
}

func TestRuntimeIgnoresLocalOriginEvent(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)
	payload, err := MarshalClipboardTextEventV1(ClipboardTextEventV1{
		Version:          EventVersionV1,
		EventID:          "evt-local",
		OriginNode:       12,
		OriginInstanceID: "instance-a",
		Text:             "hello",
	}, 64)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := rt.HandleTopicBusMessage(context.Background(), TopicBusMessage{
		Topic:   "clipboard/dev",
		Name:    ClipboardTextEventName,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("local origin returned error: %v", err)
	}
	if decision.Action != ActionIgnoredLocalOrigin {
		t.Fatalf("action = %s", decision.Action)
	}
	if len(clip.writes) != 0 {
		t.Fatalf("writes = %v", clip.writes)
	}
}

func TestRuntimeStartUpdateConfigAndResubscribe(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config:     DefaultConfig(),
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start disabled returned error: %v", err)
	}
	if len(bus.subscribed) != 0 {
		t.Fatalf("subscriptions = %v", bus.subscribed)
	}
	if err := rt.UpdateConfig(context.Background(), Config{Enabled: true, Topic: "clipboard/dev", MaxInlineBytes: 64, AutoWatch: true}); err != nil {
		t.Fatalf("UpdateConfig enable returned error: %v", err)
	}
	if clip.watches != 1 {
		t.Fatalf("watch count = %d", clip.watches)
	}
	if len(bus.subscribed) != 1 || bus.subscribed[0] != "clipboard/dev" {
		t.Fatalf("subscriptions = %v", bus.subscribed)
	}
	if err := rt.OnConnectivityRestored(context.Background()); err != nil {
		t.Fatalf("OnConnectivityRestored returned error: %v", err)
	}
	if len(bus.subscribed) != 2 {
		t.Fatalf("subscriptions after reconnect = %v", bus.subscribed)
	}
	if err := rt.UpdateConfig(context.Background(), DefaultConfig()); err != nil {
		t.Fatalf("UpdateConfig disable returned error: %v", err)
	}
	if len(bus.unsubscribed) != 1 || bus.unsubscribed[0] != "clipboard/dev" {
		t.Fatalf("unsubscriptions = %v", bus.unsubscribed)
	}
}

func TestRuntimeStartUpdateConfigWithMultipleTopics(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			MaxInlineBytes: 64,
			Topics: []TopicRoute{
				{Topic: "clipboard/a", SyncToLocal: true, SyncFromLocal: true},
				{Topic: "clipboard/b", SyncToLocal: true, SyncFromLocal: false},
			},
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if strings.Join(bus.subscribed, ",") != "clipboard/a,clipboard/b" {
		t.Fatalf("subscriptions = %v", bus.subscribed)
	}
	if err := rt.UpdateConfig(context.Background(), Config{
		Enabled:        true,
		MaxInlineBytes: 64,
		Topics: []TopicRoute{
			{Topic: "clipboard/b", SyncToLocal: false, SyncFromLocal: true},
			{Topic: "clipboard/c", SyncToLocal: true, SyncFromLocal: true},
		},
	}); err != nil {
		t.Fatalf("UpdateConfig returned error: %v", err)
	}
	if strings.Join(bus.subscribed, ",") != "clipboard/a,clipboard/b,clipboard/c" {
		t.Fatalf("subscriptions after update = %v", bus.subscribed)
	}
	if strings.Join(bus.unsubscribed, ",") != "clipboard/a" {
		t.Fatalf("unsubscriptions after update = %v", bus.unsubscribed)
	}
	if err := rt.OnConnectivityRestored(context.Background()); err != nil {
		t.Fatalf("OnConnectivityRestored returned error: %v", err)
	}
	if strings.Join(bus.subscribed, ",") != "clipboard/a,clipboard/b,clipboard/c,clipboard/b,clipboard/c" {
		t.Fatalf("subscriptions after reconnect = %v", bus.subscribed)
	}
}

func TestRuntimeReconnectCleansUpPartialMultiTopicSubscribe(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			MaxInlineBytes: 64,
			Topics: []TopicRoute{
				{Topic: "clipboard/a", SyncToLocal: true, SyncFromLocal: true},
				{Topic: "clipboard/b", SyncToLocal: true, SyncFromLocal: true},
			},
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	bus.mu.Lock()
	bus.subscribed = nil
	bus.unsubscribed = nil
	bus.subscribeErr = errors.New("subscribe failed")
	bus.subscribeErrOn = "clipboard/b"
	bus.mu.Unlock()
	if err := rt.OnConnectivityRestored(context.Background()); err == nil {
		t.Fatal("expected reconnect subscribe error")
	}
	if strings.Join(bus.subscribed, ",") != "clipboard/a" {
		t.Fatalf("subscriptions = %v", bus.subscribed)
	}
	if strings.Join(bus.unsubscribed, ",") != "clipboard/a" {
		t.Fatalf("partial cleanup unsubscriptions = %v", bus.unsubscribed)
	}
	if rt.Status().LastError == "" {
		t.Fatal("expected reconnect failure to be recorded")
	}
}

func TestRuntimeEnabledWithoutAutoWatchOnlySubscribes(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			Topic:          "clipboard/dev",
			MaxInlineBytes: 64,
			AutoWatch:      false,
			AutoApply:      false,
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	if len(bus.subscribed) != 1 {
		t.Fatalf("subscriptions = %v", bus.subscribed)
	}
	if clip.watches != 0 {
		t.Fatalf("watch count = %d", clip.watches)
	}
	status := rt.Status()
	if !status.Subscribed || status.Watching {
		t.Fatalf("status = %+v", status)
	}
}

func TestRuntimeRemoteEventPendingWhenAutoApplyDisabled(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			Topic:          "clipboard/dev",
			MaxInlineBytes: 64,
			AutoApply:      false,
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := buildRemoteMessage(t, "evt-remote", "remote text")
	decision, err := rt.HandleTopicBusMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("HandleTopicBusMessage returned error: %v", err)
	}
	if decision.Action != ActionRemotePending {
		t.Fatalf("action = %s", decision.Action)
	}
	if decision.Text != "remote text" {
		t.Fatalf("pending decision text = %q", decision.Text)
	}
	if len(clip.writes) != 0 {
		t.Fatalf("writes = %v", clip.writes)
	}
	status := rt.Status()
	if status.PendingEventID != "evt-remote" || status.PendingSize == 0 {
		t.Fatalf("pending status = %+v", status)
	}
	decision, err = rt.ApplyPending(context.Background(), "evt-remote")
	if err != nil {
		t.Fatalf("ApplyPending returned error: %v", err)
	}
	if decision.Action != ActionRemoteApplied {
		t.Fatalf("apply action = %s", decision.Action)
	}
	if decision.Text != "remote text" {
		t.Fatalf("apply decision text = %q", decision.Text)
	}
	if len(clip.writes) != 1 || clip.writes[0] != "remote text" {
		t.Fatalf("writes after apply = %v", clip.writes)
	}
	if rt.Status().PendingEventID != "" {
		t.Fatalf("pending was not cleared: %+v", rt.Status())
	}
}

func TestRuntimePendingQueueIsBounded(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			Topic:          "clipboard/dev",
			MaxInlineBytes: 64,
			AutoApply:      false,
		},
		NodeID:           12,
		InstanceID:       "instance-a",
		Clipboard:        clip,
		TopicBus:         bus,
		RecentEventLimit: 2,
		Now:              func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		msg := buildRemoteMessage(t, fmt.Sprintf("evt-pending-%d", i), fmt.Sprintf("remote text %d", i))
		if _, err := rt.HandleTopicBusMessage(context.Background(), msg); err != nil {
			t.Fatalf("HandleTopicBusMessage %d returned error: %v", i, err)
		}
	}
	pending := rt.Pending()
	if len(pending) != 2 {
		t.Fatalf("pending len = %d, pending = %+v", len(pending), pending)
	}
	if _, err := rt.ApplyPending(context.Background(), "evt-pending-1"); err == nil {
		t.Fatalf("expected evicted pending event to be missing")
	}
}

func TestRuntimeOversizeWithoutTransferIsExplicit(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)
	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "12345678901234567890123456789012345678901234567890123456789012345678901234567890"})
	if err == nil {
		t.Fatalf("expected transfer unsupported error")
	}
	if decision.Action != ActionTransferUnsupported {
		t.Fatalf("action = %s", decision.Action)
	}
	if decision.Text != "" {
		t.Fatalf("transfer unsupported leaked text = %q", decision.Text)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
}

func TestRuntimeOversizeWithTransferPublishesManifestOnly(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:          true,
			Topic:            "clipboard/dev",
			MaxInlineBytes:   4,
			DeviceLabel:      "test-device",
			AutoApply:        true,
			TransferProvider: "stream",
			TransferRef:      "source-1",
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
		NewEventID: func() (string, error) { return "transfer-local", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "oversize"})
	if err != nil {
		t.Fatalf("HandleLocalText returned error: %v", err)
	}
	if decision.Action != ActionTransferPublished {
		t.Fatalf("action = %s", decision.Action)
	}
	if decision.Text != "" {
		t.Fatalf("transfer published leaked text = %q", decision.Text)
	}
	if bus.publishCount() != 1 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
	got := bus.published[0]
	if got.Name != ClipboardTransferEventName {
		t.Fatalf("publish name = %s", got.Name)
	}
	if string(got.Payload) == "oversize" {
		t.Fatalf("manifest payload leaked body")
	}
	manifest, err := ParseClipboardTransferManifestV1(got.Payload)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.References[0].Provider != "stream" || manifest.References[0].OpaqueID != "source-1" {
		t.Fatalf("manifest refs = %+v", manifest.References)
	}
}

func TestRuntimeRemoteTransferManifestPending(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)
	digest, err := InspectTextContent("remote oversize")
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := NewClipboardTransferManifestV1(digest, []TransferReference{
		{Provider: "stream", OpaqueID: "source-remote"},
	}, BuildEventOptions{
		OriginNode:       99,
		OriginInstanceID: "remote-instance",
		MaxInlineBytes:   64,
		Now:              func() time.Time { return time.Unix(1, 0) },
		NewEventID:       func() (string, error) { return "transfer-remote", nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := MarshalClipboardTransferManifestV1(manifest)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := rt.HandleTopicBusMessage(context.Background(), TopicBusMessage{
		Topic:   "clipboard/dev",
		Name:    ClipboardTransferEventName,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("HandleTopicBusMessage returned error: %v", err)
	}
	if decision.Action != ActionTransferPending {
		t.Fatalf("action = %s", decision.Action)
	}
	if len(clip.writes) != 0 {
		t.Fatalf("writes = %v", clip.writes)
	}
}

func TestRuntimeDisableStopsWatcherEvenWhenUnsubscribeFails(t *testing.T) {
	bus := &fakeTopicBus{}
	clip := newFakeClipboard()
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			Topic:          "clipboard/dev",
			MaxInlineBytes: 64,
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := rt.Start(context.Background()); err != nil {
		t.Fatalf("Start returned error: %v", err)
	}
	bus.unsubscribeErr = errors.New("unsubscribe failed")
	if err := rt.UpdateConfig(context.Background(), DefaultConfig()); err == nil {
		t.Fatalf("expected unsubscribe error")
	}
	status := rt.Status()
	if status.Enabled || status.Subscribed {
		t.Fatalf("status after disable = %+v", status)
	}
	if status.LastError == "" {
		t.Fatalf("expected last error to be recorded")
	}
	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "after disabled"})
	if err != nil {
		t.Fatalf("local after failed unsubscribe returned error: %v", err)
	}
	if decision.Action != ActionDisabled {
		t.Fatalf("local after failed unsubscribe action = %s", decision.Action)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
}

func TestRuntimeReportsTransportAndClipboardErrors(t *testing.T) {
	bus := &fakeTopicBus{publishErr: errors.New("publish failed")}
	clip := newFakeClipboard()
	rt := newTestRuntime(t, bus, clip)
	decision, err := rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "hello"})
	if err == nil {
		t.Fatalf("expected publish error")
	}
	if decision.Action != ActionTransportFailed {
		t.Fatalf("action = %s", decision.Action)
	}
	if rt.Status().LastError == "" {
		t.Fatalf("last error was not recorded")
	}

	bus.publishErr = nil
	clip.writeErr = errors.New("write failed")
	msg := buildRemoteMessage(t, "evt-remote", "remote text")
	decision, err = rt.HandleTopicBusMessage(context.Background(), msg)
	if err == nil {
		t.Fatalf("expected clipboard write error")
	}
	if decision.Action != ActionClipboardWriteFailed {
		t.Fatalf("action = %s", decision.Action)
	}
}

func newTestRuntime(t *testing.T, bus *fakeTopicBus, clip *fakeClipboard) *Runtime {
	t.Helper()
	var seq int
	rt, err := New(Options{
		Config: Config{
			Enabled:        true,
			Topic:          "clipboard/dev",
			MaxInlineBytes: 64,
			DeviceLabel:    "test-device",
			AutoApply:      true,
		},
		NodeID:     12,
		InstanceID: "instance-a",
		Clipboard:  clip,
		TopicBus:   bus,
		Now:        func() time.Time { return time.Unix(1, 0) },
		NewEventID: func() (string, error) {
			seq++
			return "evt-local", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return rt
}

func buildRemoteMessage(t *testing.T, id string, text string) TopicBusMessage {
	t.Helper()
	return buildRemoteMessageOnTopic(t, "clipboard/dev", id, text)
}

func buildRemoteMessageOnTopic(t *testing.T, topic string, id string, text string) TopicBusMessage {
	t.Helper()
	evt, err := NewClipboardTextEventV1(text, BuildEventOptions{
		OriginNode:       99,
		OriginInstanceID: "remote-instance",
		MaxInlineBytes:   64,
		Now:              func() time.Time { return time.Unix(1, 0) },
		NewEventID:       func() (string, error) { return id, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	payload, err := MarshalClipboardTextEventV1(evt, 64)
	if err != nil {
		t.Fatal(err)
	}
	return TopicBusMessage{
		Topic:   topic,
		Name:    ClipboardTextEventName,
		Payload: payload,
	}
}
