//go:build !windows

package platform

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

type commandClipboardAdapter struct {
	readCmd      []string
	writeCmd     []string
	maxReadBytes int
	pollInterval time.Duration
}

func newClipboardAdapter(goos string, opts ClipboardOptions) (clipboard.Adapter, error) {
	if opts.MaxReadBytes <= 0 {
		return nil, fmt.Errorf("max read bytes must be positive")
	}
	switch goos {
	case "darwin":
		return &commandClipboardAdapter{
			readCmd:      []string{"pbpaste"},
			writeCmd:     []string{"pbcopy"},
			maxReadBytes: opts.MaxReadBytes,
			pollInterval: 500 * time.Millisecond,
		}, nil
	case "linux":
		return newLinuxClipboardAdapter(opts)
	default:
		return nil, clipboard.ErrUnsupported
	}
}

func newLinuxClipboardAdapter(opts ClipboardOptions) (clipboard.Adapter, error) {
	candidates := []struct {
		read  []string
		write []string
	}{
		{read: []string{"wl-paste", "--no-newline"}, write: []string{"wl-copy"}},
		{read: []string{"xclip", "-selection", "clipboard", "-out"}, write: []string{"xclip", "-selection", "clipboard", "-in"}},
		{read: []string{"xsel", "--clipboard", "--output"}, write: []string{"xsel", "--clipboard", "--input"}},
	}
	for _, candidate := range candidates {
		if _, err := exec.LookPath(candidate.read[0]); err != nil {
			continue
		}
		if _, err := exec.LookPath(candidate.write[0]); err != nil {
			continue
		}
		return &commandClipboardAdapter{
			readCmd:      candidate.read,
			writeCmd:     candidate.write,
			maxReadBytes: opts.MaxReadBytes,
			pollInterval: 500 * time.Millisecond,
		}, nil
	}
	return nil, fmt.Errorf("%w: install wl-clipboard, xclip, or xsel", clipboard.ErrUnsupported)
}

func (a *commandClipboardAdapter) ReadText(ctx context.Context) (string, error) {
	if len(a.readCmd) == 0 {
		return "", clipboard.ErrUnsupported
	}
	cmd := exec.CommandContext(ctx, a.readCmd[0], a.readCmd[1:]...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", commandError(a.readCmd[0], err, stderr.String())
	}
	if len(out) == 0 {
		return "", clipboard.ErrNoText
	}
	if len(out) > a.maxReadBytes {
		return "", fmt.Errorf("clipboard text size %d exceeds read limit %d", len(out), a.maxReadBytes)
	}
	text := string(out)
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("clipboard text is not valid utf-8")
	}
	if text == "" {
		return "", clipboard.ErrNoText
	}
	return text, nil
}

func (a *commandClipboardAdapter) WriteText(ctx context.Context, text string) error {
	if len(a.writeCmd) == 0 {
		return clipboard.ErrUnsupported
	}
	if !utf8.ValidString(text) {
		return fmt.Errorf("clipboard text is not valid utf-8")
	}
	cmd := exec.CommandContext(ctx, a.writeCmd[0], a.writeCmd[1:]...)
	cmd.Stdin = strings.NewReader(text)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return commandError(a.writeCmd[0], err, stderr.String())
	}
	return nil
}

func (a *commandClipboardAdapter) WatchText(ctx context.Context) (<-chan clipboard.TextEvent, error) {
	if len(a.readCmd) == 0 {
		return nil, clipboard.ErrUnsupported
	}
	out := make(chan clipboard.TextEvent)
	go func() {
		defer close(out)
		ticker := time.NewTicker(a.pollInterval)
		defer ticker.Stop()
		var lastText string
		var haveBaseline bool
		var lastErr string
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				text, err := a.ReadText(ctx)
				if errors.Is(err, clipboard.ErrNoText) {
					continue
				}
				if err != nil {
					if err.Error() == lastErr {
						continue
					}
					lastErr = err.Error()
					if !sendTextEvent(ctx, out, clipboard.TextEvent{Source: clipboard.SourceLocal, ObservedAt: time.Now(), Err: err}) {
						return
					}
					continue
				}
				lastErr = ""
				if !haveBaseline {
					haveBaseline = true
					lastText = text
					continue
				}
				if text == lastText {
					continue
				}
				lastText = text
				if !sendTextEvent(ctx, out, clipboard.TextEvent{Text: text, Source: clipboard.SourceLocal, ObservedAt: time.Now()}) {
					return
				}
			}
		}
	}()
	return out, nil
}

func (a *commandClipboardAdapter) Close() error {
	return nil
}

func sendTextEvent(ctx context.Context, out chan<- clipboard.TextEvent, evt clipboard.TextEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- evt:
		return true
	}
}

func commandError(name string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s failed: %w", name, err)
	}
	return fmt.Errorf("%s failed: %w: %s", name, err, stderr)
}
