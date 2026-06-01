package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
	"github.com/yttydcs/myflowhub-clipboardnode/core/myflowhub"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

type Options struct {
	Config     coreruntime.Config
	WorkDir    string
	Clipboard  clipboard.Adapter
	Transport  *myflowhub.Client
	Log        *slog.Logger
	InstanceID string
}

type Status struct {
	Connected      bool
	LoggedIn       bool
	NodeID         uint32
	HubID          uint32
	ParentEndpoint string
	Runtime        coreruntime.Status
	LastError      string
}

type Engine struct {
	log       *slog.Logger
	workDir   string
	clip      clipboard.Adapter
	transport *myflowhub.Client

	mu        sync.Mutex
	cfg       coreruntime.Config
	runtime   *coreruntime.Runtime
	cancel    context.CancelFunc
	done      chan struct{}
	started   bool
	lastErr   string
	instance  string
	decisions chan coreruntime.Decision
}

func New(opts Options) (*Engine, error) {
	cfg, err := coreruntime.NormalizeConfig(opts.Config)
	if err != nil {
		return nil, err
	}
	workDir := strings.TrimSpace(opts.WorkDir)
	if workDir == "" {
		return nil, errors.New("workDir is required")
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	transport := opts.Transport
	if transport == nil {
		transport, err = myflowhub.New(myflowhub.Options{
			WorkDir: filepath.Join(workDir, "myflowhub"),
			Log:     log,
		})
		if err != nil {
			return nil, err
		}
	}
	instanceID := strings.TrimSpace(opts.InstanceID)
	if instanceID == "" {
		instanceID, err = coreruntime.RandomEventID()
		if err != nil {
			return nil, fmt.Errorf("create instance id: %w", err)
		}
	}
	return &Engine{
		log:       log,
		workDir:   workDir,
		clip:      opts.Clipboard,
		transport: transport,
		cfg:       cfg,
		instance:  instanceID,
		decisions: make(chan coreruntime.Decision, 64),
	}, nil
}

func (e *Engine) Start(ctx context.Context) error {
	if e == nil {
		return errors.New("engine is nil")
	}
	e.mu.Lock()
	if e.started {
		e.mu.Unlock()
		return nil
	}
	cfg := e.cfg
	e.mu.Unlock()

	if err := e.transport.Connect(ctx, cfg.ParentEndpoint); err != nil {
		e.recordError(err)
		return fmt.Errorf("connect myflowhub: %w", err)
	}
	deviceID := strings.TrimSpace(cfg.DeviceLabel)
	if deviceID == "" {
		deviceID = "clipboardnode"
	}
	authState, err := e.transport.EnsureIdentity(ctx, deviceID)
	if err != nil {
		e.recordError(err)
		return fmt.Errorf("authenticate myflowhub node: %w", err)
	}
	rt, err := coreruntime.New(coreruntime.Options{
		Config:     cfg,
		NodeID:     authState.NodeID,
		InstanceID: e.instance,
		Clipboard:  e.clip,
		TopicBus:   e.transport,
	})
	if err != nil {
		e.recordError(err)
		return err
	}
	runCtx, cancel := context.WithCancel(context.Background())
	if err := rt.Start(runCtx); err != nil {
		cancel()
		e.recordError(err)
		return err
	}
	done := make(chan struct{})
	e.mu.Lock()
	e.runtime = rt
	e.cancel = cancel
	e.done = done
	e.started = true
	e.lastErr = ""
	e.mu.Unlock()
	go e.consumeRuntime(runCtx, rt, done)
	return nil
}

func (e *Engine) Stop(ctx context.Context) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	rt := e.runtime
	cancel := e.cancel
	done := e.done
	e.runtime = nil
	e.cancel = nil
	e.done = nil
	e.started = false
	e.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if done != nil {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	var rtErr error
	if rt != nil {
		rtErr = rt.Stop(ctx)
	}
	var closeErr error
	if e.transport != nil {
		closeErr = e.transport.Close()
	}
	if rtErr != nil {
		e.recordError(rtErr)
		return rtErr
	}
	if closeErr != nil {
		e.recordError(closeErr)
		return closeErr
	}
	return nil
}

func (e *Engine) UpdateConfig(ctx context.Context, cfg coreruntime.Config) error {
	if e == nil {
		return errors.New("engine is nil")
	}
	cfg, err := coreruntime.NormalizeConfig(cfg)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.cfg = cfg
	rt := e.runtime
	e.mu.Unlock()
	if rt != nil {
		if err := rt.UpdateConfig(ctx, cfg); err != nil {
			e.recordError(err)
			return err
		}
	}
	return nil
}

func (e *Engine) SendText(ctx context.Context, text string) (coreruntime.Decision, error) {
	if e == nil {
		return coreruntime.Decision{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	rt := e.runtime
	e.mu.Unlock()
	if rt == nil {
		return coreruntime.Decision{}, errors.New("engine is not started")
	}
	decision, err := rt.HandleLocalText(ctx, clipboard.TextEvent{
		Text:       text,
		Source:     clipboard.SourceLocal,
		ObservedAt: time.Now(),
	})
	if err != nil {
		e.recordError(err)
	}
	return decision, err
}

func (e *Engine) ReadClipboard(ctx context.Context) (coreruntime.Decision, error) {
	if e == nil {
		return coreruntime.Decision{}, errors.New("engine is nil")
	}
	if e.clip == nil {
		return coreruntime.Decision{}, errors.New("clipboard adapter is required")
	}
	text, err := e.clip.ReadText(ctx)
	if err != nil {
		e.recordError(err)
		return coreruntime.Decision{}, fmt.Errorf("read clipboard text: %w", err)
	}
	return e.SendText(ctx, text)
}

func (e *Engine) ApplyPending(ctx context.Context, eventID string) (coreruntime.Decision, error) {
	if e == nil {
		return coreruntime.Decision{}, errors.New("engine is nil")
	}
	e.mu.Lock()
	rt := e.runtime
	e.mu.Unlock()
	if rt == nil {
		return coreruntime.Decision{}, errors.New("engine is not started")
	}
	decision, err := rt.ApplyPending(ctx, eventID)
	if err != nil {
		e.recordError(err)
	}
	return decision, err
}

func (e *Engine) Decisions() <-chan coreruntime.Decision {
	if e == nil {
		return nil
	}
	return e.decisions
}

func (e *Engine) Status() Status {
	if e == nil {
		return Status{}
	}
	e.mu.Lock()
	cfg := e.cfg
	rt := e.runtime
	lastErr := e.lastErr
	e.mu.Unlock()
	transportStatus := e.transport.Status()
	out := Status{
		Connected:      transportStatus.Connected,
		LoggedIn:       transportStatus.Auth.LoggedIn,
		NodeID:         transportStatus.Auth.NodeID,
		HubID:          transportStatus.Auth.HubID,
		ParentEndpoint: cfg.ParentEndpoint,
		LastError:      lastErr,
	}
	if out.LastError == "" {
		out.LastError = transportStatus.LastError
	}
	if rt != nil {
		out.Runtime = rt.Status()
	}
	return out
}

func (e *Engine) consumeRuntime(ctx context.Context, rt *coreruntime.Runtime, done chan<- struct{}) {
	defer close(done)
	events := e.transport.Events()
	decisions := rt.Decisions()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-events:
			if !ok {
				return
			}
			if _, err := rt.HandleTopicBusMessage(ctx, msg); err != nil {
				e.recordError(err)
				if e.log != nil {
					e.log.Warn("clipboard topicbus message failed",
						"topic", msg.Topic,
						"name", msg.Name,
						"err", err.Error(),
					)
				}
			}
		case decision, ok := <-decisions:
			if !ok {
				decisions = nil
				continue
			}
			e.emitDecision(decision)
		}
	}
}

func (e *Engine) emitDecision(decision coreruntime.Decision) {
	if e == nil || e.decisions == nil || decision.Action == "" {
		return
	}
	select {
	case e.decisions <- decision:
	default:
		e.recordError(errors.New("engine decision queue full"))
	}
}

func (e *Engine) recordError(err error) {
	if e == nil || err == nil {
		return
	}
	e.mu.Lock()
	e.lastErr = err.Error()
	e.mu.Unlock()
}
