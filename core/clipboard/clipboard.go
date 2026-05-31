package clipboard

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNoText      = errors.New("clipboard does not contain text")
	ErrUnsupported = errors.New("clipboard adapter is unsupported on this platform")
)

type Source string

const (
	SourceLocal  Source = "local"
	SourceRemote Source = "remote"
)

type TextEvent struct {
	Text       string
	Source     Source
	ObservedAt time.Time
	Err        error
}

type Adapter interface {
	ReadText(ctx context.Context) (string, error)
	WriteText(ctx context.Context, text string) error
	WatchText(ctx context.Context) (<-chan TextEvent, error)
	Close() error
}
