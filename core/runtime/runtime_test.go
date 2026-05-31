package runtime

import (
	"context"
	"encoding/json"
	"errors"
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
	publishErr     error
	unsubscribeErr error
}

func (f *fakeTopicBus) Subscribe(_ context.Context, topic string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.subscribeErr != nil {
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
	if bus.publishCount() != 1 {
		t.Fatalf("publish count = %d", bus.publishCount())
	}
	got := bus.published[0]
	if got.Topic != "clipboard/dev" || got.Name != ClipboardTextEventName {
		t.Fatalf("unexpected topicbus message: %+v", got)
	}
	var evt ClipboardTextEventV1
	if err := json.Unmarshal(got.Payload, &evt); err != nil {
		t.Fatal(err)
	}
	if evt.Text != "hello" {
		t.Fatalf("event text = %q", evt.Text)
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

	decision, err = rt.HandleLocalText(context.Background(), clipboard.TextEvent{Text: "remote text"})
	if err != nil {
		t.Fatalf("loop local returned error: %v", err)
	}
	if decision.Action != ActionIgnoredLoop {
		t.Fatalf("loop action = %s", decision.Action)
	}
	if bus.publishCount() != 0 {
		t.Fatalf("publish count = %d", bus.publishCount())
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
		ContentType:      ContentTypeTextPlain,
		Encoding:         EncodingUTF8,
		Size:             len("hello"),
		SHA256:           HashText("hello"),
		Text:             "hello",
		TS:               time.Unix(1, 0).UnixMilli(),
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
	if err := rt.UpdateConfig(context.Background(), Config{Enabled: true, Topic: "clipboard/dev", MaxInlineBytes: 64}); err != nil {
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
		Topic:   "clipboard/dev",
		Name:    ClipboardTextEventName,
		Payload: payload,
	}
}
