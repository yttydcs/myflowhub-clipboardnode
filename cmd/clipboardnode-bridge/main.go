package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub-clipboardnode/bridge"
	"github.com/yttydcs/myflowhub-clipboardnode/core/configstore"
	"github.com/yttydcs/myflowhub-clipboardnode/core/engine"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
	"github.com/yttydcs/myflowhub-clipboardnode/platform"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	defaultPath, err := configPath()
	if err != nil {
		return err
	}
	configFile := flag.String("config", defaultPath, "path to ClipboardNode JSON config")
	webListen := flag.String("web-listen", "", "optional localhost HTTP/SSE bridge address, for example 127.0.0.1:18291")
	webToken := flag.String("web-token", "", "required token for HTTP/SSE bridge requests; generated when empty")
	flag.Parse()

	store, err := configstore.New(*configFile)
	if err != nil {
		return err
	}
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	adapter, err := platform.NewClipboardAdapter(platform.ClipboardOptions{MaxReadBytes: cfg.MaxInlineBytes})
	if err != nil {
		return fmt.Errorf("initialize clipboard adapter: %w", err)
	}
	defer adapter.Close()
	eng, err := engine.New(engine.Options{
		Config:    cfg,
		WorkDir:   filepath.Dir(*configFile),
		Clipboard: adapter,
		Log:       slog.New(slog.NewTextHandler(os.Stderr, nil)),
	})
	if err != nil {
		return err
	}
	host := &stdioHost{
		store: store,
		cfg:   cfg,
		eng:   eng,
		out:   os.Stdout,
	}
	if strings.TrimSpace(*webListen) != "" {
		token := strings.TrimSpace(*webToken)
		if token == "" {
			token, err = generateToken()
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "clipboardnode web bridge token: %s\n", token)
		}
		return host.serveWeb(context.Background(), strings.TrimSpace(*webListen), token)
	}
	return host.serve(context.Background(), os.Stdin)
}

type stdioHost struct {
	store *configstore.Store
	cfg   coreruntime.Config
	eng   *engine.Engine
	out   io.Writer
	mu    sync.Mutex
	cmdMu sync.Mutex
}

func (h *stdioHost) serve(ctx context.Context, in io.Reader) error {
	if h == nil || h.eng == nil {
		return errors.New("bridge host is not initialized")
	}
	h.emitStatus("", true, "")
	go h.forwardDecisions(ctx)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var cmd bridge.EngineCommand
		if err := json.Unmarshal([]byte(line), &cmd); err != nil {
			h.emitError("", fmt.Errorf("decode command: %w", err))
			continue
		}
		shutdown, _ := h.handle(ctx, cmd)
		if shutdown {
			return nil
		}
	}
	return scanner.Err()
}

func (h *stdioHost) handle(ctx context.Context, cmd bridge.EngineCommand) (bool, error) {
	h.cmdMu.Lock()
	defer h.cmdMu.Unlock()
	switch cmd.Action {
	case bridge.ActionConnect:
		err := h.eng.Start(ctx)
		h.emitStatus(cmd.ID, err == nil, errorString(err))
		return false, err
	case bridge.ActionSetConfig:
		var settings bridge.Settings
		if err := json.Unmarshal(cmd.Data, &settings); err != nil {
			h.emitError(cmd.ID, fmt.Errorf("decode settings: %w", err))
			return false, err
		}
		cfg, err := coreruntime.NormalizeConfig(configFromSettings(settings))
		if err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		if err := h.store.Save(cfg); err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		if err := h.eng.UpdateConfig(ctx, cfg); err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		h.cfg = cfg
		h.emitStatus(cmd.ID, true, "")
	case bridge.ActionSendText:
		var req struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(cmd.Data, &req); err != nil {
			h.emitError(cmd.ID, fmt.Errorf("decode send_text: %w", err))
			return false, err
		}
		if _, err := h.eng.SendText(ctx, req.Text); err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		h.emitCommandOK(cmd.ID)
		h.emitStatus("", true, "")
	case bridge.ActionReadClipboard:
		if _, err := h.eng.ReadClipboard(ctx); err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		h.emitCommandOK(cmd.ID)
		h.emitStatus("", true, "")
	case bridge.ActionApplyEvent:
		var req struct {
			EventID string `json:"event_id"`
		}
		if err := json.Unmarshal(cmd.Data, &req); err != nil {
			h.emitError(cmd.ID, fmt.Errorf("decode apply_event: %w", err))
			return false, err
		}
		if _, err := h.eng.ApplyPending(ctx, req.EventID); err != nil {
			h.emitError(cmd.ID, err)
			return false, err
		}
		h.emitCommandOK(cmd.ID)
		h.emitStatus("", true, "")
	case bridge.ActionClearRecent:
		h.emit(bridge.EngineEvent{ID: cmd.ID, Name: bridge.EventActivityUpdated, OK: true})
	case bridge.ActionShutdown:
		err := h.eng.Stop(context.Background())
		h.emitStatus(cmd.ID, err == nil, errorString(err))
		return true, err
	default:
		err := fmt.Errorf("unsupported action %q", cmd.Action)
		h.emitError(cmd.ID, err)
		return false, err
	}
	return false, nil
}

func (h *stdioHost) serveWeb(ctx context.Context, listen string, token string) error {
	if err := validateLoopbackListen(listen); err != nil {
		return err
	}
	bus := newEventBus()
	h.out = bus
	h.emitStatus("", true, "")
	go h.forwardDecisions(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	mux.HandleFunc("/events", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		bus.serveSSE(w, r)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		data, _ := json.Marshal(h.status())
		_, _ = w.Write(data)
	})
	mux.HandleFunc("/command", func(w http.ResponseWriter, r *http.Request) {
		writeCORS(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if !authorized(r, token) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		defer r.Body.Close()
		var cmd bridge.EngineCommand
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&cmd); err != nil {
			http.Error(w, fmt.Sprintf("decode command: %v", err), http.StatusBadRequest)
			return
		}
		shutdown, err := h.handle(r.Context(), cmd)
		response := map[string]any{
			"accepted": true,
			"ok":       err == nil,
			"error":    errorString(err),
			"status":   h.status(),
		}
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusAccepted)
		}
		_ = json.NewEncoder(w).Encode(response)
		if shutdown {
			go func() {
				_ = h.eng.Stop(context.Background())
			}()
		}
	})

	server := &http.Server{
		Addr:    listen,
		Handler: mux,
	}
	fmt.Fprintf(os.Stderr, "clipboardnode web bridge listening on http://%s\n", listen)
	return server.ListenAndServe()
}

func (h *stdioHost) emitStatus(id string, ok bool, errMsg string) {
	data, _ := json.Marshal(h.statusWithError(errMsg))
	h.emit(bridge.EngineEvent{ID: id, Name: bridge.EventStatusChanged, Data: data, OK: ok, Error: errMsg})
}

func (h *stdioHost) status() bridge.Status {
	return h.statusWithError("")
}

func (h *stdioHost) statusWithError(errMsg string) bridge.Status {
	status := h.eng.Status()
	cfg := h.cfg
	if status.Runtime.Topic != "" {
		cfg.Topic = status.Runtime.Topic
		cfg.Enabled = status.Runtime.Enabled
		cfg.AutoWatch = status.Runtime.AutoWatch
		cfg.AutoApply = status.Runtime.AutoApply
	}
	if cfg.ParentEndpoint == "" {
		cfg.ParentEndpoint = status.ParentEndpoint
	}
	lastErr := errMsg
	if lastErr == "" {
		lastErr = status.LastError
	}
	return bridge.Status{
		Connected:        status.Connected,
		LoggedIn:         status.LoggedIn,
		NodeID:           status.NodeID,
		HubID:            status.HubID,
		ParentEndpoint:   cfg.ParentEndpoint,
		Enabled:          cfg.Enabled,
		Topic:            cfg.Topic,
		DeviceID:         cfg.DeviceID,
		DisplayName:      cfg.DisplayName,
		DeviceLabel:      cfg.DisplayName,
		AutoWatch:        cfg.AutoWatch,
		AutoApply:        cfg.AutoApply,
		TransferProvider: cfg.TransferProvider,
		TransferRef:      cfg.TransferRef,
		Started:          status.Runtime.Started,
		Subscribed:       status.Runtime.Subscribed,
		Watching:         status.Runtime.Watching,
		PendingEventID:   status.Runtime.PendingEventID,
		PendingSize:      status.Runtime.PendingSize,
		PendingHash:      status.Runtime.PendingHashPrefix,
		LastAction:       string(status.Runtime.LastAction),
		LastEventID:      status.Runtime.LastEventID,
		LastSize:         status.Runtime.LastSize,
		LastHashPrefix:   status.Runtime.LastHash,
		LastError:        lastErr,
	}
}

func (h *stdioHost) emitActivity(id string, decision coreruntime.Decision) {
	if decision.Action == "" {
		return
	}
	data, _ := json.Marshal(bridge.Activity{
		ID:          decision.EventID,
		Kind:        activityKind(decision.Action),
		Title:       string(decision.Action),
		Detail:      "TopicBus",
		ByteSize:    decision.Size,
		HashPrefix:  decision.HashPrefix,
		TimestampMS: h.eng.Status().Runtime.LastUpdated.UnixMilli(),
	})
	h.emit(bridge.EngineEvent{ID: id, Name: bridge.EventActivityUpdated, Data: data, OK: true})
}

func (h *stdioHost) emitTransferIfNeeded(id string, decision coreruntime.Decision) {
	var state string
	switch decision.Action {
	case coreruntime.ActionTransferPublished:
		state = "manifest_published"
	case coreruntime.ActionTransferPending:
		state = "manifest_pending"
	case coreruntime.ActionTransferUnsupported:
		state = "unsupported"
	default:
		return
	}
	data, _ := json.Marshal(bridge.Transfer{
		ID:         decision.EventID,
		State:      state,
		ByteSize:   decision.Size,
		HashPrefix: decision.HashPrefix,
		Detail:     "clipboard.transfer.v1",
	})
	h.emit(bridge.EngineEvent{ID: id, Name: bridge.EventTransferUpdate, Data: data, OK: true})
}

func (h *stdioHost) emitCommandOK(id string) {
	if strings.TrimSpace(id) == "" {
		return
	}
	h.emit(bridge.EngineEvent{ID: id, Name: bridge.EventStatusChanged, OK: true})
}

func (h *stdioHost) emitError(id string, err error) {
	h.emit(bridge.EngineEvent{ID: id, Name: bridge.EventError, OK: false, Error: errorString(err)})
}

func (h *stdioHost) emit(evt bridge.EngineEvent) {
	raw, _ := json.Marshal(evt)
	h.mu.Lock()
	defer h.mu.Unlock()
	fmt.Fprintln(h.out, string(raw))
}

type eventBus struct {
	mu      sync.Mutex
	clients map[chan string]struct{}
}

func newEventBus() *eventBus {
	return &eventBus{clients: map[chan string]struct{}{}}
}

func (b *eventBus) Write(payload []byte) (int, error) {
	for _, line := range strings.Split(strings.TrimSpace(string(payload)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		b.broadcast(line)
	}
	return len(payload), nil
}

func (b *eventBus) broadcast(line string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- line:
		default:
		}
	}
}

func (b *eventBus) serveSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch := make(chan string, 32)
	b.mu.Lock()
	b.clients[ch] = struct{}{}
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.clients, ch)
		b.mu.Unlock()
		close(ch)
	}()
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case line := <-ch:
			_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
			flusher.Flush()
		}
	}
}

func writeCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-ClipboardNode-Token")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

func authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	return r.Header.Get("X-ClipboardNode-Token") == token || r.URL.Query().Get("token") == token
}

func validateLoopbackListen(listen string) error {
	host, port, err := net.SplitHostPort(listen)
	if err != nil {
		return fmt.Errorf("invalid web-listen address: %w", err)
	}
	if strings.TrimSpace(port) == "" {
		return errors.New("web-listen port is required")
	}
	if host == "" {
		return errors.New("web-listen host must be explicit loopback")
	}
	if strings.EqualFold(host, "localhost") {
		return nil
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("web-listen must bind to a loopback address, got %q", host)
	}
	return nil
}

func generateToken() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("generate web bridge token: %w", err)
	}
	return hex.EncodeToString(buf[:]), nil
}

func (h *stdioHost) forwardDecisions(ctx context.Context) {
	decisions := h.eng.Decisions()
	for {
		select {
		case <-ctx.Done():
			return
		case decision, ok := <-decisions:
			if !ok {
				return
			}
			h.emitActivity("", decision)
			h.emitTransferIfNeeded("", decision)
			h.emitStatus("", true, "")
		}
	}
}

func configFromSettings(settings bridge.Settings) coreruntime.Config {
	return coreruntime.Config{
		Enabled:          settings.Enabled,
		ParentEndpoint:   settings.ParentEndpoint,
		Topic:            settings.Topic,
		MaxInlineBytes:   settings.MaxInlineBytes,
		DeviceID:         settings.DeviceID,
		DisplayName:      settings.DisplayName,
		DeviceLabel:      settings.DeviceLabel,
		AutoWatch:        settings.AutoWatch,
		AutoApply:        settings.AutoApply,
		HistoryRetention: settings.HistoryRetention,
		TransferProvider: settings.TransferProvider,
		TransferRef:      settings.TransferRef,
	}
}

func activityKind(action coreruntime.Action) string {
	switch action {
	case coreruntime.ActionLocalPublished:
		return "published"
	case coreruntime.ActionRemoteApplied:
		return "applied"
	case coreruntime.ActionRemotePending, coreruntime.ActionTransferPending, coreruntime.ActionTransferPublished:
		return "pending"
	case coreruntime.ActionValidationFailed, coreruntime.ActionTransportFailed, coreruntime.ActionClipboardWriteFailed:
		return "error"
	default:
		return "ignored"
	}
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "MyFlowHub", "ClipboardNode", "config.json"), nil
}
