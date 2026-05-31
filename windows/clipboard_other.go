//go:build !windows

package windows

import (
	"context"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

type Options struct {
	MaxReadBytes int
	PollInterval time.Duration
	OpenTimeout  time.Duration
}

type ClipboardAdapter struct{}

func NewClipboardAdapter(Options) (*ClipboardAdapter, error) {
	return nil, clipboard.ErrUnsupported
}

func (a *ClipboardAdapter) ReadText(context.Context) (string, error) {
	return "", clipboard.ErrUnsupported
}

func (a *ClipboardAdapter) WriteText(context.Context, string) error {
	return clipboard.ErrUnsupported
}

func (a *ClipboardAdapter) WatchText(context.Context) (<-chan clipboard.TextEvent, error) {
	return nil, clipboard.ErrUnsupported
}

func (a *ClipboardAdapter) Close() error {
	return nil
}
