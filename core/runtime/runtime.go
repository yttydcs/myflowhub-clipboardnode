package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

type TopicBusMessage struct {
	Topic   string
	Name    string
	Payload []byte
}

type TopicBusClient interface {
	Subscribe(ctx context.Context, topic string) error
	Unsubscribe(ctx context.Context, topic string) error
	Publish(ctx context.Context, topic string, name string, payload []byte) error
}

type Options struct {
	Config            Config
	NodeID            uint32
	InstanceID        string
	Clipboard         clipboard.Adapter
	TopicBus          TopicBusClient
	Now               func() time.Time
	NewEventID        func() (string, error)
	RecentEventLimit  int
	SuppressHashLimit int
}

type Action string

const (
	ActionDisabled             Action = "disabled"
	ActionIgnoredTopic         Action = "ignored_topic"
	ActionIgnoredName          Action = "ignored_name"
	ActionIgnoredLocalOrigin   Action = "ignored_local_origin"
	ActionIgnoredDuplicate     Action = "ignored_duplicate"
	ActionIgnoredLoop          Action = "ignored_loop"
	ActionIgnoredUnchanged     Action = "ignored_unchanged"
	ActionLocalPublished       Action = "local_published"
	ActionRemoteApplied        Action = "remote_applied"
	ActionValidationFailed     Action = "validation_failed"
	ActionTransportFailed      Action = "transport_failed"
	ActionClipboardWriteFailed Action = "clipboard_write_failed"
)

type Decision struct {
	Action     Action
	EventID    string
	Size       int
	HashPrefix string
}

type Status struct {
	Enabled     bool
	Topic       string
	Started     bool
	Subscribed  bool
	LastAction  Action
	LastEventID string
	LastSize    int
	LastHash    string
	LastError   string
	LastUpdated time.Time
}

type Runtime struct {
	mu         sync.Mutex
	cfg        Config
	nodeID     uint32
	instanceID string
	clipboard  clipboard.Adapter
	topicBus   TopicBusClient
	now        func() time.Time
	newEventID func() (string, error)

	recentEvents   *boundedStringSet
	suppressHashes *boundedStringSet
	lastLocalHash  string

	started     bool
	subscribed  bool
	watching    bool
	runCtx      context.Context
	cancel      context.CancelFunc
	watchCancel context.CancelFunc
	status      Status
}

func New(opts Options) (*Runtime, error) {
	cfg, err := NormalizeConfig(opts.Config)
	if err != nil {
		return nil, err
	}
	if opts.NodeID == 0 {
		return nil, fmt.Errorf("node id is required")
	}
	if opts.InstanceID == "" {
		return nil, fmt.Errorf("instance id is required")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	newEventID := opts.NewEventID
	if newEventID == nil {
		newEventID = RandomEventID
	}
	recentLimit := opts.RecentEventLimit
	if recentLimit == 0 {
		recentLimit = defaultRecentLimit
	}
	suppressLimit := opts.SuppressHashLimit
	if suppressLimit == 0 {
		suppressLimit = defaultSuppressLimit
	}
	rt := &Runtime{
		cfg:            cfg,
		nodeID:         opts.NodeID,
		instanceID:     opts.InstanceID,
		clipboard:      opts.Clipboard,
		topicBus:       opts.TopicBus,
		now:            now,
		newEventID:     newEventID,
		recentEvents:   newBoundedStringSet(recentLimit),
		suppressHashes: newBoundedStringSet(suppressLimit),
	}
	rt.status = Status{
		Enabled:     cfg.Enabled,
		Topic:       cfg.Topic,
		LastUpdated: now(),
	}
	return rt, nil
}

func (r *Runtime) Start(ctx context.Context) error {
	r.mu.Lock()
	if r.started {
		r.mu.Unlock()
		return nil
	}
	cfg := r.cfg
	runCtx, cancel := context.WithCancel(ctx)
	r.runCtx = runCtx
	r.cancel = cancel
	if !cfg.Enabled {
		r.started = true
		r.status.Started = true
		r.status.LastAction = ActionDisabled
		r.status.LastUpdated = r.now()
		r.mu.Unlock()
		return nil
	}
	if r.topicBus == nil {
		cancel()
		r.runCtx = nil
		r.cancel = nil
		r.mu.Unlock()
		return fmt.Errorf("topicbus client is required when clipboard sync is enabled")
	}
	if r.clipboard == nil {
		cancel()
		r.runCtx = nil
		r.cancel = nil
		r.mu.Unlock()
		return fmt.Errorf("clipboard adapter is required when clipboard sync is enabled")
	}
	r.mu.Unlock()

	if err := r.topicBus.Subscribe(runCtx, cfg.Topic); err != nil {
		cancel()
		r.clearStartState()
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return fmt.Errorf("subscribe clipboard topic: %w", err)
	}
	watchCtx, watchCancel := context.WithCancel(runCtx)
	events, err := r.clipboard.WatchText(watchCtx)
	if err != nil {
		_ = r.topicBus.Unsubscribe(context.Background(), cfg.Topic)
		cancel()
		r.clearStartState()
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return fmt.Errorf("watch clipboard text: %w", err)
	}
	r.mu.Lock()
	r.started = true
	r.subscribed = true
	r.watching = true
	r.watchCancel = watchCancel
	r.status.Started = true
	r.status.Subscribed = true
	r.status.LastUpdated = r.now()
	r.mu.Unlock()

	go r.runClipboardWatcher(watchCtx, events)
	return nil
}

func (r *Runtime) Stop(ctx context.Context) error {
	r.mu.Lock()
	cancel := r.cancel
	watchCancel := r.watchCancel
	cfg := r.cfg
	subscribed := r.subscribed
	r.started = false
	r.subscribed = false
	r.watching = false
	r.runCtx = nil
	r.cancel = nil
	r.watchCancel = nil
	r.status.Started = false
	r.status.Subscribed = false
	r.status.LastUpdated = r.now()
	r.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if watchCancel != nil {
		watchCancel()
	}
	var unsubscribeErr error
	if subscribed && r.topicBus != nil {
		unsubscribeErr = r.topicBus.Unsubscribe(ctx, cfg.Topic)
	}
	var closeErr error
	if r.clipboard != nil {
		closeErr = r.clipboard.Close()
	}
	if unsubscribeErr != nil {
		return fmt.Errorf("unsubscribe clipboard topic: %w", unsubscribeErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close clipboard adapter: %w", closeErr)
	}
	return nil
}

func (r *Runtime) UpdateConfig(ctx context.Context, next Config) error {
	next, err := NormalizeConfig(next)
	if err != nil {
		return err
	}

	r.mu.Lock()
	previous := r.cfg
	started := r.started
	watching := r.watching
	runCtx := r.runCtx
	watchCancel := r.watchCancel
	topicBus := r.topicBus
	clip := r.clipboard
	r.mu.Unlock()

	if !started {
		r.mu.Lock()
		r.cfg = next
		r.status.Enabled = next.Enabled
		r.status.Topic = next.Topic
		r.status.LastUpdated = r.now()
		r.mu.Unlock()
		return nil
	}
	if next.Enabled && topicBus == nil {
		return fmt.Errorf("topicbus client is required when clipboard sync is enabled")
	}
	if next.Enabled && clip == nil {
		return fmt.Errorf("clipboard adapter is required when clipboard sync is enabled")
	}

	if previous.Enabled && !next.Enabled {
		if watchCancel != nil {
			watchCancel()
		}
		r.mu.Lock()
		r.cfg = next
		r.subscribed = false
		r.watching = false
		r.watchCancel = nil
		r.status.Enabled = next.Enabled
		r.status.Topic = next.Topic
		r.status.Subscribed = false
		r.status.LastAction = ActionDisabled
		r.status.LastUpdated = r.now()
		r.status.LastError = ""
		r.mu.Unlock()

		var unsubscribeErr error
		if topicBus != nil {
			unsubscribeErr = topicBus.Unsubscribe(ctx, previous.Topic)
		}
		r.mu.Lock()
		if unsubscribeErr != nil {
			r.status.LastError = unsubscribeErr.Error()
		}
		r.mu.Unlock()
		if unsubscribeErr != nil {
			return fmt.Errorf("unsubscribe clipboard topic: %w", unsubscribeErr)
		}
		return nil
	}

	needsSubscribe := next.Enabled && (!previous.Enabled || previous.Topic != next.Topic)
	needsUnsubscribe := previous.Enabled && (!next.Enabled || previous.Topic != next.Topic)
	if needsSubscribe {
		if err := topicBus.Subscribe(ctx, next.Topic); err != nil {
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("subscribe clipboard topic: %w", err)
		}
	}

	var events <-chan clipboard.TextEvent
	var watcherCtx context.Context
	var newWatchCancel context.CancelFunc
	if next.Enabled && !watching {
		watcherCtx, newWatchCancel = context.WithCancel(runCtx)
		events, err = clip.WatchText(watcherCtx)
		if err != nil {
			newWatchCancel()
			if needsSubscribe {
				_ = topicBus.Unsubscribe(context.Background(), next.Topic)
			}
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("watch clipboard text: %w", err)
		}
	}
	if needsUnsubscribe {
		if err := topicBus.Unsubscribe(ctx, previous.Topic); err != nil {
			if needsSubscribe {
				_ = topicBus.Unsubscribe(context.Background(), next.Topic)
			}
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("unsubscribe clipboard topic: %w", err)
		}
	}

	r.mu.Lock()
	r.cfg = next
	r.subscribed = next.Enabled
	if !next.Enabled && watchCancel != nil {
		watchCancel()
		r.watching = false
		r.watchCancel = nil
	}
	r.status.Enabled = next.Enabled
	r.status.Topic = next.Topic
	r.status.Subscribed = next.Enabled
	r.status.LastUpdated = r.now()
	if events != nil {
		r.watching = true
		r.watchCancel = newWatchCancel
	}
	r.mu.Unlock()
	if events != nil {
		go r.runClipboardWatcher(watcherCtx, events)
	}
	return nil
}

func (r *Runtime) OnConnectivityRestored(ctx context.Context) error {
	r.mu.Lock()
	cfg := r.cfg
	started := r.started
	topicBus := r.topicBus
	r.mu.Unlock()
	if !started || !cfg.Enabled {
		return nil
	}
	if topicBus == nil {
		return fmt.Errorf("topicbus client is required")
	}
	if err := topicBus.Subscribe(ctx, cfg.Topic); err != nil {
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return fmt.Errorf("resubscribe clipboard topic: %w", err)
	}
	r.mu.Lock()
	r.subscribed = true
	r.status.Subscribed = true
	r.status.LastUpdated = r.now()
	r.mu.Unlock()
	return nil
}

func (r *Runtime) HandleLocalText(ctx context.Context, evt clipboard.TextEvent) (Decision, error) {
	r.mu.Lock()
	cfg := r.cfg
	if !cfg.Enabled {
		decision := Decision{Action: ActionDisabled}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	topicBus := r.topicBus
	nodeID := r.nodeID
	instanceID := r.instanceID
	deviceLabel := cfg.DeviceLabel
	now := r.now
	newEventID := r.newEventID
	r.mu.Unlock()

	if topicBus == nil {
		err := fmt.Errorf("topicbus client is required")
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return Decision{Action: ActionTransportFailed}, err
	}
	digest, err := InspectText(evt.Text, cfg.MaxInlineBytes)
	if err != nil {
		r.recordFailure(ActionValidationFailed, "", 0, "", err)
		return Decision{Action: ActionValidationFailed}, err
	}
	hashPrefix := hashPrefix(digest.SHA256)

	r.mu.Lock()
	if r.suppressHashes.Consume(digest.SHA256) {
		decision := Decision{Action: ActionIgnoredLoop, Size: digest.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if digest.SHA256 == r.lastLocalHash {
		decision := Decision{Action: ActionIgnoredUnchanged, Size: digest.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	r.mu.Unlock()

	out, err := newClipboardTextEventV1WithDigest(evt.Text, digest, BuildEventOptions{
		OriginNode:       nodeID,
		OriginInstanceID: instanceID,
		OriginDevice:     deviceLabel,
		MaxInlineBytes:   cfg.MaxInlineBytes,
		Now:              now,
		NewEventID:       newEventID,
	})
	if err != nil {
		r.recordFailure(ActionValidationFailed, "", digest.Size, hashPrefix, err)
		return Decision{Action: ActionValidationFailed, Size: digest.Size, HashPrefix: hashPrefix}, err
	}
	payload, err := MarshalClipboardTextEventV1(out, cfg.MaxInlineBytes)
	if err != nil {
		r.recordFailure(ActionValidationFailed, out.EventID, digest.Size, hashPrefix, err)
		return Decision{Action: ActionValidationFailed, EventID: out.EventID, Size: digest.Size, HashPrefix: hashPrefix}, err
	}
	if err := topicBus.Publish(ctx, cfg.Topic, ClipboardTextEventName, payload); err != nil {
		r.recordFailure(ActionTransportFailed, out.EventID, digest.Size, hashPrefix, err)
		return Decision{Action: ActionTransportFailed, EventID: out.EventID, Size: digest.Size, HashPrefix: hashPrefix}, fmt.Errorf("publish clipboard event: %w", err)
	}

	decision := Decision{Action: ActionLocalPublished, EventID: out.EventID, Size: digest.Size, HashPrefix: hashPrefix}
	r.mu.Lock()
	r.lastLocalHash = digest.SHA256
	r.recentEvents.Add(out.EventID)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) HandleTopicBusMessage(ctx context.Context, msg TopicBusMessage) (Decision, error) {
	r.mu.Lock()
	cfg := r.cfg
	if !cfg.Enabled {
		decision := Decision{Action: ActionDisabled}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if msg.Topic != cfg.Topic {
		decision := Decision{Action: ActionIgnoredTopic}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if msg.Name != ClipboardTextEventName {
		decision := Decision{Action: ActionIgnoredName}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	clip := r.clipboard
	nodeID := r.nodeID
	instanceID := r.instanceID
	r.mu.Unlock()

	if clip == nil {
		err := fmt.Errorf("clipboard adapter is required")
		r.recordFailure(ActionClipboardWriteFailed, "", 0, "", err)
		return Decision{Action: ActionClipboardWriteFailed}, err
	}
	in, err := ParseClipboardTextEventV1(msg.Payload, cfg.MaxInlineBytes)
	if err != nil {
		r.recordFailure(ActionValidationFailed, "", 0, "", err)
		return Decision{Action: ActionValidationFailed}, err
	}
	hashPrefix := hashPrefix(in.SHA256)

	r.mu.Lock()
	if in.IsLocalOrigin(nodeID, instanceID) {
		decision := Decision{Action: ActionIgnoredLocalOrigin, EventID: in.EventID, Size: in.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if r.recentEvents.Contains(in.EventID) {
		decision := Decision{Action: ActionIgnoredDuplicate, EventID: in.EventID, Size: in.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	r.mu.Unlock()

	if err := clip.WriteText(ctx, in.Text); err != nil {
		r.recordFailure(ActionClipboardWriteFailed, in.EventID, in.Size, hashPrefix, err)
		return Decision{Action: ActionClipboardWriteFailed, EventID: in.EventID, Size: in.Size, HashPrefix: hashPrefix}, fmt.Errorf("write clipboard text: %w", err)
	}

	decision := Decision{Action: ActionRemoteApplied, EventID: in.EventID, Size: in.Size, HashPrefix: hashPrefix}
	r.mu.Lock()
	r.recentEvents.Add(in.EventID)
	r.suppressHashes.Add(in.SHA256)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) Status() Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.status
}

func (r *Runtime) runClipboardWatcher(ctx context.Context, events <-chan clipboard.TextEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-events:
			if !ok {
				return
			}
			if evt.Err != nil {
				r.recordFailure(ActionValidationFailed, "", 0, "", evt.Err)
				continue
			}
			_, _ = r.HandleLocalText(ctx, evt)
		}
	}
}

func (r *Runtime) clearStartState() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.started = false
	r.subscribed = false
	r.watching = false
	r.runCtx = nil
	r.cancel = nil
	r.watchCancel = nil
	r.status.Started = false
	r.status.Subscribed = false
	r.status.LastUpdated = r.now()
}

func (r *Runtime) recordFailure(action Action, eventID string, size int, hash string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	decision := Decision{Action: action, EventID: eventID, Size: size, HashPrefix: hash}
	r.recordDecisionLocked(decision)
	if err != nil {
		r.status.LastError = err.Error()
	}
}

func (r *Runtime) recordDecisionLocked(decision Decision) {
	r.status.Enabled = r.cfg.Enabled
	r.status.Topic = r.cfg.Topic
	r.status.Started = r.started
	r.status.Subscribed = r.subscribed
	r.status.LastAction = decision.Action
	r.status.LastEventID = decision.EventID
	r.status.LastSize = decision.Size
	r.status.LastHash = decision.HashPrefix
	r.status.LastUpdated = r.now()
	if decision.Action != ActionValidationFailed && decision.Action != ActionTransportFailed && decision.Action != ActionClipboardWriteFailed {
		r.status.LastError = ""
	}
}

func hashPrefix(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}
