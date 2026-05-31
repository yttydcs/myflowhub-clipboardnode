//go:build windows

package windows

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002

	defaultPollInterval = 250 * time.Millisecond
	defaultOpenTimeout  = 250 * time.Millisecond
	defaultRetryDelay   = 10 * time.Millisecond
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procEmptyClipboard             = user32.NewProc("EmptyClipboard")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procSetClipboardData           = user32.NewProc("SetClipboardData")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	procGlobalAlloc                = kernel32.NewProc("GlobalAlloc")
	procGlobalFree                 = kernel32.NewProc("GlobalFree")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
)

type Options struct {
	MaxReadBytes int
	PollInterval time.Duration
	OpenTimeout  time.Duration
}

type ClipboardAdapter struct {
	maxReadBytes int
	pollInterval time.Duration
	openTimeout  time.Duration
}

func NewClipboardAdapter(opts Options) (*ClipboardAdapter, error) {
	if opts.MaxReadBytes <= 0 {
		return nil, fmt.Errorf("max read bytes must be positive")
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = defaultPollInterval
	}
	if opts.OpenTimeout <= 0 {
		opts.OpenTimeout = defaultOpenTimeout
	}
	return &ClipboardAdapter{
		maxReadBytes: opts.MaxReadBytes,
		pollInterval: opts.PollInterval,
		openTimeout:  opts.OpenTimeout,
	}, nil
}

func (a *ClipboardAdapter) ReadText(ctx context.Context) (string, error) {
	if err := a.openClipboard(ctx); err != nil {
		return "", err
	}
	defer procCloseClipboard.Call()

	available, _, _ := procIsClipboardFormatAvailable.Call(cfUnicodeText)
	if available == 0 {
		return "", clipboard.ErrNoText
	}
	handle, _, callErr := procGetClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", win32Error("GetClipboardData", callErr)
	}
	size, _, callErr := procGlobalSize.Call(handle)
	if size == 0 {
		return "", win32Error("GlobalSize", callErr)
	}
	data, _, callErr := procGlobalLock.Call(handle)
	if data == 0 {
		return "", win32Error("GlobalLock", callErr)
	}
	defer procGlobalUnlock.Call(handle)

	maxUnits := int(size / 2)
	limitUnits := a.maxReadBytes + 1
	if maxUnits > limitUnits {
		maxUnits = limitUnits
	}
	utf16Text := unsafe.Slice((*uint16)(unsafe.Pointer(data)), maxUnits)
	end := -1
	for i, unit := range utf16Text {
		if unit == 0 {
			end = i
			break
		}
	}
	if end < 0 {
		return "", fmt.Errorf("clipboard text exceeds read limit %d", a.maxReadBytes)
	}
	text := syscall.UTF16ToString(utf16Text[:end+1])
	if !utf8.ValidString(text) {
		return "", fmt.Errorf("clipboard text is not valid utf-8")
	}
	if len(text) > a.maxReadBytes {
		return "", fmt.Errorf("clipboard text size %d exceeds read limit %d", len(text), a.maxReadBytes)
	}
	return text, nil
}

func (a *ClipboardAdapter) WriteText(ctx context.Context, text string) error {
	if !utf8.ValidString(text) {
		return fmt.Errorf("clipboard text is not valid utf-8")
	}
	utf16Text, err := syscall.UTF16FromString(text)
	if err != nil {
		return fmt.Errorf("encode clipboard text: %w", err)
	}
	size := uintptr(len(utf16Text) * 2)
	handle, _, callErr := procGlobalAlloc.Call(gmemMoveable, size)
	if handle == 0 {
		return win32Error("GlobalAlloc", callErr)
	}
	owned := true
	defer func() {
		if owned {
			procGlobalFree.Call(handle)
		}
	}()

	data, _, callErr := procGlobalLock.Call(handle)
	if data == 0 {
		return win32Error("GlobalLock", callErr)
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(data)), len(utf16Text)), utf16Text)
	procGlobalUnlock.Call(handle)

	if err := a.openClipboard(ctx); err != nil {
		return err
	}
	defer procCloseClipboard.Call()
	if result, _, callErr := procEmptyClipboard.Call(); result == 0 {
		return win32Error("EmptyClipboard", callErr)
	}
	if result, _, callErr := procSetClipboardData.Call(cfUnicodeText, handle); result == 0 {
		return win32Error("SetClipboardData", callErr)
	}
	owned = false
	return nil
}

func (a *ClipboardAdapter) WatchText(ctx context.Context) (<-chan clipboard.TextEvent, error) {
	out := make(chan clipboard.TextEvent)
	go func() {
		defer close(out)
		ticker := time.NewTicker(a.pollInterval)
		defer ticker.Stop()

		var lastSequence uintptr
		var lastErrorHash [32]byte
		var haveBaseline bool
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				sequence, _, _ := procGetClipboardSequenceNumber.Call()
				if haveBaseline && sequence == lastSequence {
					continue
				}
				lastSequence = sequence
				if !haveBaseline {
					haveBaseline = true
					continue
				}
				text, err := a.ReadText(ctx)
				if errors.Is(err, clipboard.ErrNoText) {
					lastErrorHash = [32]byte{}
					continue
				}
				if err != nil {
					currentErrorHash := sha256.Sum256([]byte(err.Error()))
					if currentErrorHash == lastErrorHash {
						continue
					}
					lastErrorHash = currentErrorHash
					if !sendTextEvent(ctx, out, clipboard.TextEvent{Source: clipboard.SourceLocal, ObservedAt: time.Now(), Err: err}) {
						return
					}
					continue
				}
				lastErrorHash = [32]byte{}
				if !sendTextEvent(ctx, out, clipboard.TextEvent{Text: text, Source: clipboard.SourceLocal, ObservedAt: time.Now()}) {
					return
				}
			}
		}
	}()
	return out, nil
}

func (a *ClipboardAdapter) Close() error {
	return nil
}

func (a *ClipboardAdapter) openClipboard(ctx context.Context) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, a.openTimeout)
	defer cancel()
	ticker := time.NewTicker(defaultRetryDelay)
	defer ticker.Stop()
	for {
		if result, _, _ := procOpenClipboard.Call(0); result != 0 {
			return nil
		}
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("open clipboard: %w", timeoutCtx.Err())
		case <-ticker.C:
		}
	}
}

func sendTextEvent(ctx context.Context, out chan<- clipboard.TextEvent, evt clipboard.TextEvent) bool {
	select {
	case <-ctx.Done():
		return false
	case out <- evt:
		return true
	}
}

func win32Error(operation string, err error) error {
	if err == nil || errors.Is(err, syscall.Errno(0)) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}
