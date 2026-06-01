package nodemobile

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
	"github.com/yttydcs/myflowhub-clipboardnode/core/engine"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

var (
	mu      sync.Mutex
	current *engine.Engine
	clip    *manualClipboard
	lastErr string
)

type manualClipboard struct {
	mu   sync.Mutex
	text string
}

func (m *manualClipboard) ReadText(context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if strings.TrimSpace(m.text) == "" {
		return "", clipboard.ErrNoText
	}
	return m.text, nil
}

func (m *manualClipboard) WriteText(_ context.Context, text string) error {
	m.mu.Lock()
	m.text = text
	m.mu.Unlock()
	return nil
}

func (m *manualClipboard) WatchText(context.Context) (<-chan clipboard.TextEvent, error) {
	return nil, clipboard.ErrUnsupported
}

func (m *manualClipboard) Close() error {
	return nil
}

func Start(configJSON string, workDir string) (string, error) {
	cfg, err := parseConfig(configJSON)
	if err != nil {
		setLastError(err)
		return "", err
	}
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		err := errors.New("workDir is required")
		setLastError(err)
		return "", err
	}
	manual := &manualClipboard{}
	eng, err := engine.New(engine.Options{
		Config:    cfg,
		WorkDir:   workDir,
		Clipboard: manual,
		Log:       slog.Default(),
	})
	if err != nil {
		setLastError(err)
		return "", err
	}
	if err := eng.Start(context.Background()); err != nil {
		setLastError(err)
		return "", err
	}
	mu.Lock()
	if current != nil {
		_ = current.Stop(context.Background())
	}
	current = eng
	clip = manual
	mu.Unlock()
	return statusJSON(eng), nil
}

func Stop() (string, error) {
	mu.Lock()
	eng := current
	current = nil
	clip = nil
	mu.Unlock()
	if eng == nil {
		return statusJSON(nil), nil
	}
	if err := eng.Stop(context.Background()); err != nil {
		setLastError(err)
		return "", err
	}
	return statusJSON(nil), nil
}

func UpdateConfig(configJSON string) (string, error) {
	cfg, err := parseConfig(configJSON)
	if err != nil {
		setLastError(err)
		return "", err
	}
	mu.Lock()
	eng := current
	mu.Unlock()
	if eng == nil {
		err := errors.New("engine is not started")
		setLastError(err)
		return "", err
	}
	if err := eng.UpdateConfig(context.Background(), cfg); err != nil {
		setLastError(err)
		return "", err
	}
	setLastError(nil)
	return statusJSON(eng), nil
}

func SendText(text string) (string, error) {
	mu.Lock()
	eng := current
	mu.Unlock()
	if eng == nil {
		err := errors.New("engine is not started")
		setLastError(err)
		return "", err
	}
	decision, err := eng.SendText(context.Background(), text)
	if err != nil {
		setLastError(err)
		return "", err
	}
	raw, _ := json.Marshal(decision)
	return string(raw), nil
}

func SetClipboardText(text string) string {
	mu.Lock()
	manual := clip
	mu.Unlock()
	if manual == nil {
		manual = &manualClipboard{}
		mu.Lock()
		clip = manual
		mu.Unlock()
	}
	_ = manual.WriteText(context.Background(), text)
	return Status()
}

func ReadClipboard() (string, error) {
	mu.Lock()
	eng := current
	mu.Unlock()
	if eng == nil {
		err := errors.New("engine is not started")
		setLastError(err)
		return "", err
	}
	decision, err := eng.ReadClipboard(context.Background())
	if err != nil {
		setLastError(err)
		return "", err
	}
	raw, _ := json.Marshal(decision)
	return string(raw), nil
}

func ApplyEvent(eventID string) (string, error) {
	mu.Lock()
	eng := current
	mu.Unlock()
	if eng == nil {
		err := errors.New("engine is not started")
		setLastError(err)
		return "", err
	}
	decision, err := eng.ApplyPending(context.Background(), eventID)
	if err != nil {
		setLastError(err)
		return "", err
	}
	raw, _ := json.Marshal(decision)
	return string(raw), nil
}

func Status() string {
	mu.Lock()
	eng := current
	mu.Unlock()
	return statusJSON(eng)
}

func parseConfig(raw string) (coreruntime.Config, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return coreruntime.DefaultConfig(), nil
	}
	var cfg coreruntime.Config
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return coreruntime.Config{}, err
	}
	return coreruntime.NormalizeConfig(cfg)
}

func statusJSON(eng *engine.Engine) string {
	if eng == nil {
		raw, _ := json.Marshal(map[string]any{
			"running":    false,
			"last_error": lastErr,
		})
		return string(raw)
	}
	raw, _ := json.Marshal(eng.Status())
	return string(raw)
}

func setLastError(err error) {
	mu.Lock()
	defer mu.Unlock()
	if err == nil {
		lastErr = ""
		return
	}
	lastErr = err.Error()
}
