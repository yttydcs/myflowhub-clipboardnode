package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
	DecisionBuffer    int
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
	ActionIgnoredLocalPolicy   Action = "ignored_local_policy"
	ActionIgnoredTopicPolicy   Action = "ignored_topic_policy"
	ActionLocalPublished       Action = "local_published"
	ActionTransferPublished    Action = "transfer_published"
	ActionTransferPending      Action = "transfer_pending"
	ActionTransferUnsupported  Action = "transfer_unsupported"
	ActionRemotePending        Action = "remote_pending"
	ActionRemoteApplied        Action = "remote_applied"
	ActionValidationFailed     Action = "validation_failed"
	ActionTransportFailed      Action = "transport_failed"
	ActionClipboardWriteFailed Action = "clipboard_write_failed"
)

type Decision struct {
	Action     Action
	EventID    string
	Topic      string
	Size       int
	HashPrefix string
	Text       string `json:"-"`
}

type Status struct {
	Enabled           bool
	Topic             string
	Topics            []TopicRoute
	AutoWatch         bool
	AutoApply         bool
	Started           bool
	Subscribed        bool
	Watching          bool
	PendingEventID    string
	PendingTopic      string
	PendingSize       int
	PendingHashPrefix string
	LastAction        Action
	LastEventID       string
	LastSize          int
	LastHash          string
	LastError         string
	LastUpdated       time.Time
}

type PendingEvent struct {
	EventID      string
	Topic        string
	Size         int
	HashPrefix   string
	OriginNode   uint32
	OriginDevice string
	ReceivedAt   time.Time
}

type pendingClipboardText struct {
	event      ClipboardTextEventV1
	topic      string
	hashPrefix string
	receivedAt time.Time
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
	pendingRemote  map[string]pendingClipboardText
	pendingOrder   []string
	pendingLimit   int
	decisions      chan Decision

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
	decisionBuffer := opts.DecisionBuffer
	if decisionBuffer <= 0 {
		decisionBuffer = 64
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
		pendingRemote:  make(map[string]pendingClipboardText),
		pendingLimit:   recentLimit,
		decisions:      make(chan Decision, decisionBuffer),
	}
	rt.status = Status{
		Enabled:     cfg.Enabled,
		Topic:       cfg.Topic,
		Topics:      CloneTopicRoutes(cfg.Topics),
		AutoWatch:   cfg.AutoWatch,
		AutoApply:   cfg.AutoApply,
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
	if (cfg.AutoWatch || cfg.AutoApply) && r.clipboard == nil {
		cancel()
		r.runCtx = nil
		r.cancel = nil
		r.mu.Unlock()
		return fmt.Errorf("clipboard adapter is required when clipboard sync is enabled")
	}
	topics := subscriptionTopics(cfg)
	r.mu.Unlock()

	subscribedTopics, err := subscribeTopics(runCtx, r.topicBus, topics)
	if err != nil {
		_ = unsubscribeTopics(context.Background(), r.topicBus, subscribedTopics)
		cancel()
		r.clearStartState()
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return fmt.Errorf("subscribe clipboard topics: %w", err)
	}
	var watchCtx context.Context
	var watchCancel context.CancelFunc
	var events <-chan clipboard.TextEvent
	if cfg.AutoWatch {
		watchCtx, watchCancel = context.WithCancel(runCtx)
		var err error
		events, err = r.clipboard.WatchText(watchCtx)
		if err != nil {
			watchCancel()
			_ = unsubscribeTopics(context.Background(), r.topicBus, subscribedTopics)
			cancel()
			r.clearStartState()
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("watch clipboard text: %w", err)
		}
	}
	r.mu.Lock()
	r.started = true
	r.subscribed = true
	r.watching = cfg.AutoWatch
	r.watchCancel = watchCancel
	r.status.Started = true
	r.status.Subscribed = true
	r.status.Watching = cfg.AutoWatch
	r.status.LastUpdated = r.now()
	r.mu.Unlock()

	if events != nil {
		go r.runClipboardWatcher(watchCtx, events)
	}
	return nil
}

func (r *Runtime) Decisions() <-chan Decision {
	if r == nil {
		return nil
	}
	return r.decisions
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
		unsubscribeErr = unsubscribeTopics(ctx, r.topicBus, subscriptionTopics(cfg))
	}
	var closeErr error
	if r.clipboard != nil {
		closeErr = r.clipboard.Close()
	}
	if unsubscribeErr != nil {
		return fmt.Errorf("unsubscribe clipboard topics: %w", unsubscribeErr)
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
		r.status.Topics = CloneTopicRoutes(next.Topics)
		r.status.AutoWatch = next.AutoWatch
		r.status.AutoApply = next.AutoApply
		r.status.LastUpdated = r.now()
		r.mu.Unlock()
		return nil
	}
	if next.Enabled && topicBus == nil {
		return fmt.Errorf("topicbus client is required when clipboard sync is enabled")
	}
	if next.Enabled && (next.AutoWatch || next.AutoApply) && clip == nil {
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
		r.status.Topics = CloneTopicRoutes(next.Topics)
		r.status.AutoWatch = next.AutoWatch
		r.status.AutoApply = next.AutoApply
		r.status.Subscribed = false
		r.status.Watching = false
		r.status.LastAction = ActionDisabled
		r.pendingRemote = make(map[string]pendingClipboardText)
		r.pendingOrder = nil
		r.status.LastUpdated = r.now()
		r.status.LastError = ""
		r.mu.Unlock()

		var unsubscribeErr error
		if topicBus != nil {
			unsubscribeErr = unsubscribeTopics(ctx, topicBus, subscriptionTopics(previous))
		}
		r.mu.Lock()
		if unsubscribeErr != nil {
			r.status.LastError = unsubscribeErr.Error()
		}
		r.mu.Unlock()
		if unsubscribeErr != nil {
			return fmt.Errorf("unsubscribe clipboard topics: %w", unsubscribeErr)
		}
		return nil
	}

	needsSubscribe := topicsToSubscribe(previous, next)
	needsUnsubscribe := topicsToUnsubscribe(previous, next)
	if len(needsSubscribe) > 0 {
		subscribedTopics, err := subscribeTopics(ctx, topicBus, needsSubscribe)
		if err != nil {
			_ = unsubscribeTopics(context.Background(), topicBus, subscribedTopics)
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("subscribe clipboard topics: %w", err)
		}
	}

	var events <-chan clipboard.TextEvent
	var watcherCtx context.Context
	var newWatchCancel context.CancelFunc
	shouldStopWatcher := watching && (!next.Enabled || !next.AutoWatch)
	if shouldStopWatcher && watchCancel != nil {
		watchCancel()
	}
	if next.Enabled && next.AutoWatch && !watching {
		watcherCtx, newWatchCancel = context.WithCancel(runCtx)
		events, err = clip.WatchText(watcherCtx)
		if err != nil {
			newWatchCancel()
			if len(needsSubscribe) > 0 {
				_ = unsubscribeTopics(context.Background(), topicBus, needsSubscribe)
			}
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("watch clipboard text: %w", err)
		}
	}
	if len(needsUnsubscribe) > 0 {
		if err := unsubscribeTopics(ctx, topicBus, needsUnsubscribe); err != nil {
			if len(needsSubscribe) > 0 {
				_ = unsubscribeTopics(context.Background(), topicBus, needsSubscribe)
			}
			r.recordFailure(ActionTransportFailed, "", 0, "", err)
			return fmt.Errorf("unsubscribe clipboard topics: %w", err)
		}
	}

	r.mu.Lock()
	r.cfg = next
	r.subscribed = next.Enabled
	if shouldStopWatcher {
		r.watching = false
		r.watchCancel = nil
	}
	r.status.Enabled = next.Enabled
	r.status.Topic = next.Topic
	r.status.Topics = CloneTopicRoutes(next.Topics)
	r.status.AutoWatch = next.AutoWatch
	r.status.AutoApply = next.AutoApply
	r.status.Subscribed = next.Enabled
	r.status.Watching = r.watching
	r.status.LastUpdated = r.now()
	if events != nil {
		r.watching = true
		r.watchCancel = newWatchCancel
		r.status.Watching = true
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
	subscribedTopics, err := subscribeTopics(ctx, topicBus, subscriptionTopics(cfg))
	if err != nil {
		_ = unsubscribeTopics(context.Background(), topicBus, subscribedTopics)
		r.recordFailure(ActionTransportFailed, "", 0, "", err)
		return fmt.Errorf("resubscribe clipboard topics: %w", err)
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
		contentDigest, inspectErr := InspectTextContent(evt.Text)
		if inspectErr != nil {
			r.recordFailure(ActionValidationFailed, "", 0, "", err)
			return Decision{Action: ActionValidationFailed}, err
		}
		return r.publishTransferManifest(ctx, contentDigest, err)
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

	publishTargets := publishTopics(cfg)
	if len(publishTargets) == 0 {
		decision := Decision{Action: ActionIgnoredLocalPolicy, Size: digest.Size, HashPrefix: hashPrefix}
		r.mu.Lock()
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}

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
	for _, topic := range publishTargets {
		if err := topicBus.Publish(ctx, topic, ClipboardTextEventName, payload); err != nil {
			decision := Decision{Action: ActionTransportFailed, EventID: out.EventID, Topic: topic, Size: digest.Size, HashPrefix: hashPrefix}
			r.recordFailureDecision(decision, err)
			return decision, fmt.Errorf("publish clipboard event to %q: %w", topic, err)
		}
	}

	decision := Decision{Action: ActionLocalPublished, EventID: out.EventID, Topic: topicListLabel(publishTargets), Size: digest.Size, HashPrefix: hashPrefix, Text: evt.Text}
	r.mu.Lock()
	r.lastLocalHash = digest.SHA256
	r.recentEvents.Add(out.EventID)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) ApplyPending(ctx context.Context, eventID string) (Decision, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return Decision{}, fmt.Errorf("event_id is required")
	}
	r.mu.Lock()
	pending, ok := r.pendingRemote[eventID]
	if !ok {
		r.mu.Unlock()
		return Decision{}, fmt.Errorf("pending clipboard event %q not found", eventID)
	}
	clip := r.clipboard
	r.mu.Unlock()

	if clip == nil {
		err := fmt.Errorf("clipboard adapter is required")
		decision := Decision{Action: ActionClipboardWriteFailed, EventID: pending.event.EventID, Topic: pending.topic, Size: pending.event.Size, HashPrefix: pending.hashPrefix}
		r.recordFailureDecision(decision, err)
		return decision, err
	}
	if err := clip.WriteText(ctx, pending.event.Text); err != nil {
		decision := Decision{Action: ActionClipboardWriteFailed, EventID: pending.event.EventID, Topic: pending.topic, Size: pending.event.Size, HashPrefix: pending.hashPrefix}
		r.recordFailureDecision(decision, err)
		return decision, fmt.Errorf("write clipboard text: %w", err)
	}

	decision := Decision{Action: ActionRemoteApplied, EventID: pending.event.EventID, Topic: pending.topic, Size: pending.event.Size, HashPrefix: pending.hashPrefix, Text: pending.event.Text}
	r.mu.Lock()
	r.deletePendingLocked(eventID)
	r.recentEvents.Add(pending.event.EventID)
	r.suppressHashes.Add(pending.event.SHA256)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) Pending() []PendingEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]PendingEvent, 0, len(r.pendingRemote))
	for _, pending := range r.pendingRemote {
		out = append(out, PendingEvent{
			EventID:      pending.event.EventID,
			Topic:        pending.topic,
			Size:         pending.event.Size,
			HashPrefix:   pending.hashPrefix,
			OriginNode:   pending.event.OriginNode,
			OriginDevice: pending.event.OriginDevice,
			ReceivedAt:   pending.receivedAt,
		})
	}
	return out
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
	route, ok := topicRouteFor(cfg, msg.Topic)
	if !ok {
		decision := Decision{Action: ActionIgnoredTopic, Topic: msg.Topic}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if !route.SyncToLocal {
		decision := Decision{Action: ActionIgnoredTopicPolicy, Topic: msg.Topic}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if msg.Name != ClipboardTextEventName && msg.Name != ClipboardTransferEventName {
		decision := Decision{Action: ActionIgnoredName, Topic: msg.Topic}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	clip := r.clipboard
	nodeID := r.nodeID
	instanceID := r.instanceID
	autoApply := cfg.AutoApply
	r.mu.Unlock()
	if msg.Name == ClipboardTransferEventName {
		return r.handleTransferManifest(msg.Topic, msg.Payload, cfg.MaxInlineBytes, nodeID, instanceID)
	}
	in, err := ParseClipboardTextEventV1(msg.Payload, cfg.MaxInlineBytes)
	if err != nil {
		decision := Decision{Action: ActionValidationFailed, Topic: msg.Topic}
		r.recordFailureDecision(decision, err)
		return decision, err
	}
	hashPrefix := hashPrefix(in.SHA256)

	r.mu.Lock()
	if in.IsLocalOrigin(nodeID, instanceID) {
		decision := Decision{Action: ActionIgnoredLocalOrigin, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if r.recentEvents.Contains(in.EventID) {
		decision := Decision{Action: ActionIgnoredDuplicate, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if _, ok := r.pendingRemote[in.EventID]; ok {
		decision := Decision{Action: ActionIgnoredDuplicate, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	if !autoApply {
		decision := Decision{Action: ActionRemotePending, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix, Text: in.Text}
		r.addPendingLocked(pendingClipboardText{event: in, topic: msg.Topic, hashPrefix: hashPrefix, receivedAt: r.now()})
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	r.mu.Unlock()

	if clip == nil {
		err := fmt.Errorf("clipboard adapter is required")
		decision := Decision{Action: ActionClipboardWriteFailed, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix}
		r.recordFailureDecision(decision, err)
		return decision, err
	}
	if err := clip.WriteText(ctx, in.Text); err != nil {
		decision := Decision{Action: ActionClipboardWriteFailed, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix}
		r.recordFailureDecision(decision, err)
		return decision, fmt.Errorf("write clipboard text: %w", err)
	}

	decision := Decision{Action: ActionRemoteApplied, EventID: in.EventID, Topic: msg.Topic, Size: in.Size, HashPrefix: hashPrefix, Text: in.Text}
	r.mu.Lock()
	r.recentEvents.Add(in.EventID)
	r.suppressHashes.Add(in.SHA256)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) publishTransferManifest(ctx context.Context, digest TextDigest, cause error) (Decision, error) {
	r.mu.Lock()
	cfg := r.cfg
	topicBus := r.topicBus
	nodeID := r.nodeID
	instanceID := r.instanceID
	deviceLabel := cfg.DeviceLabel
	now := r.now
	newEventID := r.newEventID
	r.mu.Unlock()
	hashPrefix := hashPrefix(digest.SHA256)
	if cfg.TransferProvider == "" || cfg.TransferRef == "" {
		err := fmt.Errorf("clipboard text requires transfer manifest but transfer is not configured: %w", cause)
		r.recordFailure(ActionTransferUnsupported, "", digest.Size, hashPrefix, err)
		return Decision{Action: ActionTransferUnsupported, Size: digest.Size, HashPrefix: hashPrefix}, err
	}
	if topicBus == nil {
		err := fmt.Errorf("topicbus client is required")
		r.recordFailure(ActionTransportFailed, "", digest.Size, hashPrefix, err)
		return Decision{Action: ActionTransportFailed, Size: digest.Size, HashPrefix: hashPrefix}, err
	}
	publishTargets := publishTopics(cfg)
	if len(publishTargets) == 0 {
		decision := Decision{Action: ActionIgnoredLocalPolicy, Size: digest.Size, HashPrefix: hashPrefix}
		r.mu.Lock()
		r.recordDecisionLocked(decision)
		r.mu.Unlock()
		return decision, nil
	}
	manifest, err := NewClipboardTransferManifestV1(digest, []TransferReference{
		{Provider: cfg.TransferProvider, OpaqueID: cfg.TransferRef},
	}, BuildEventOptions{
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
	payload, err := MarshalClipboardTransferManifestV1(manifest)
	if err != nil {
		r.recordFailure(ActionValidationFailed, manifest.EventID, digest.Size, hashPrefix, err)
		return Decision{Action: ActionValidationFailed, EventID: manifest.EventID, Size: digest.Size, HashPrefix: hashPrefix}, err
	}
	for _, topic := range publishTargets {
		if err := topicBus.Publish(ctx, topic, ClipboardTransferEventName, payload); err != nil {
			decision := Decision{Action: ActionTransportFailed, EventID: manifest.EventID, Topic: topic, Size: digest.Size, HashPrefix: hashPrefix}
			r.recordFailureDecision(decision, err)
			return decision, fmt.Errorf("publish clipboard transfer manifest to %q: %w", topic, err)
		}
	}
	decision := Decision{Action: ActionTransferPublished, EventID: manifest.EventID, Topic: topicListLabel(publishTargets), Size: digest.Size, HashPrefix: hashPrefix}
	r.mu.Lock()
	r.lastLocalHash = digest.SHA256
	r.recentEvents.Add(manifest.EventID)
	r.recordDecisionLocked(decision)
	r.mu.Unlock()
	return decision, nil
}

func (r *Runtime) handleTransferManifest(topic string, payload []byte, maxInlineBytes int, nodeID uint32, instanceID string) (Decision, error) {
	_ = maxInlineBytes
	manifest, err := ParseClipboardTransferManifestV1(payload)
	if err != nil {
		decision := Decision{Action: ActionValidationFailed, Topic: topic}
		r.recordFailureDecision(decision, err)
		return decision, err
	}
	hashPrefix := hashPrefix(manifest.SHA256)
	r.mu.Lock()
	defer r.mu.Unlock()
	if manifest.IsLocalOrigin(nodeID, instanceID) {
		decision := Decision{Action: ActionIgnoredLocalOrigin, EventID: manifest.EventID, Topic: topic, Size: manifest.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		return decision, nil
	}
	if r.recentEvents.Contains(manifest.EventID) {
		decision := Decision{Action: ActionIgnoredDuplicate, EventID: manifest.EventID, Topic: topic, Size: manifest.Size, HashPrefix: hashPrefix}
		r.recordDecisionLocked(decision)
		return decision, nil
	}
	r.recentEvents.Add(manifest.EventID)
	decision := Decision{Action: ActionTransferPending, EventID: manifest.EventID, Topic: topic, Size: manifest.Size, HashPrefix: hashPrefix}
	r.recordDecisionLocked(decision)
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
	r.recordFailureDecision(Decision{Action: action, EventID: eventID, Size: size, HashPrefix: hash}, err)
}

func (r *Runtime) recordFailureDecision(decision Decision, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.recordDecisionLocked(decision)
	if err != nil {
		r.status.LastError = err.Error()
	}
}

func (r *Runtime) recordDecisionLocked(decision Decision) {
	r.status.Enabled = r.cfg.Enabled
	r.status.Topic = r.cfg.Topic
	r.status.Topics = CloneTopicRoutes(r.cfg.Topics)
	r.status.AutoWatch = r.cfg.AutoWatch
	r.status.AutoApply = r.cfg.AutoApply
	r.status.Started = r.started
	r.status.Subscribed = r.subscribed
	r.status.Watching = r.watching
	r.status.LastAction = decision.Action
	r.status.LastEventID = decision.EventID
	r.status.LastSize = decision.Size
	r.status.LastHash = decision.HashPrefix
	r.status.PendingEventID = ""
	r.status.PendingTopic = ""
	r.status.PendingSize = 0
	r.status.PendingHashPrefix = ""
	for i := len(r.pendingOrder) - 1; i >= 0; i-- {
		eventID := r.pendingOrder[i]
		pending, ok := r.pendingRemote[eventID]
		if !ok {
			continue
		}
		r.status.PendingEventID = pending.event.EventID
		r.status.PendingTopic = pending.topic
		r.status.PendingSize = pending.event.Size
		r.status.PendingHashPrefix = pending.hashPrefix
		break
	}
	r.status.LastUpdated = r.now()
	if decision.Action != ActionValidationFailed && decision.Action != ActionTransportFailed && decision.Action != ActionClipboardWriteFailed && decision.Action != ActionTransferUnsupported {
		r.status.LastError = ""
	}
	r.emitDecisionLocked(decision)
}

func (r *Runtime) addPendingLocked(pending pendingClipboardText) {
	eventID := pending.event.EventID
	if eventID == "" {
		return
	}
	if _, ok := r.pendingRemote[eventID]; !ok {
		r.pendingOrder = append(r.pendingOrder, eventID)
	}
	r.pendingRemote[eventID] = pending
	for len(r.pendingRemote) > r.pendingLimit && len(r.pendingOrder) > 0 {
		oldest := r.pendingOrder[0]
		r.pendingOrder = r.pendingOrder[1:]
		if _, ok := r.pendingRemote[oldest]; ok {
			delete(r.pendingRemote, oldest)
			r.recentEvents.Add(oldest)
		}
	}
	if len(r.pendingOrder) > r.pendingLimit*2 {
		kept := r.pendingOrder[:0]
		for _, id := range r.pendingOrder {
			if _, ok := r.pendingRemote[id]; ok {
				kept = append(kept, id)
			}
		}
		r.pendingOrder = kept
	}
}

func (r *Runtime) deletePendingLocked(eventID string) {
	delete(r.pendingRemote, eventID)
	for i, id := range r.pendingOrder {
		if id == eventID {
			copy(r.pendingOrder[i:], r.pendingOrder[i+1:])
			r.pendingOrder = r.pendingOrder[:len(r.pendingOrder)-1]
			return
		}
	}
}

func (r *Runtime) emitDecisionLocked(decision Decision) {
	if r.decisions == nil {
		return
	}
	select {
	case r.decisions <- decision:
	default:
	}
}

func hashPrefix(hash string) string {
	if len(hash) <= 12 {
		return hash
	}
	return hash[:12]
}

func subscriptionTopics(cfg Config) []string {
	topics := make([]string, 0, len(cfg.Topics))
	for _, route := range cfg.Topics {
		if route.Topic != "" {
			topics = append(topics, route.Topic)
		}
	}
	return topics
}

func publishTopics(cfg Config) []string {
	topics := make([]string, 0, len(cfg.Topics))
	for _, route := range cfg.Topics {
		if route.SyncFromLocal && route.Topic != "" {
			topics = append(topics, route.Topic)
		}
	}
	return topics
}

func topicRouteFor(cfg Config, topic string) (TopicRoute, bool) {
	topic = strings.TrimSpace(topic)
	for _, route := range cfg.Topics {
		if route.Topic == topic {
			return route, true
		}
	}
	return TopicRoute{}, false
}

func topicsToSubscribe(previous Config, next Config) []string {
	if !next.Enabled {
		return nil
	}
	nextTopics := subscriptionTopics(next)
	if !previous.Enabled {
		return nextTopics
	}
	previousSet := topicSet(subscriptionTopics(previous))
	out := make([]string, 0, len(nextTopics))
	for _, topic := range nextTopics {
		if _, ok := previousSet[topic]; !ok {
			out = append(out, topic)
		}
	}
	return out
}

func topicsToUnsubscribe(previous Config, next Config) []string {
	if !previous.Enabled {
		return nil
	}
	previousTopics := subscriptionTopics(previous)
	if !next.Enabled {
		return previousTopics
	}
	nextSet := topicSet(subscriptionTopics(next))
	out := make([]string, 0, len(previousTopics))
	for _, topic := range previousTopics {
		if _, ok := nextSet[topic]; !ok {
			out = append(out, topic)
		}
	}
	return out
}

func topicSet(topics []string) map[string]struct{} {
	out := make(map[string]struct{}, len(topics))
	for _, topic := range topics {
		out[topic] = struct{}{}
	}
	return out
}

func subscribeTopics(ctx context.Context, client TopicBusClient, topics []string) ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("topicbus client is required")
	}
	subscribed := make([]string, 0, len(topics))
	for _, topic := range topics {
		if err := client.Subscribe(ctx, topic); err != nil {
			return subscribed, fmt.Errorf("subscribe %q: %w", topic, err)
		}
		subscribed = append(subscribed, topic)
	}
	return subscribed, nil
}

func unsubscribeTopics(ctx context.Context, client TopicBusClient, topics []string) error {
	if client == nil || len(topics) == 0 {
		return nil
	}
	var errs []error
	for _, topic := range topics {
		if err := client.Unsubscribe(ctx, topic); err != nil {
			errs = append(errs, fmt.Errorf("unsubscribe %q: %w", topic, err))
		}
	}
	return errors.Join(errs...)
}

func topicListLabel(topics []string) string {
	return strings.Join(topics, ", ")
}
