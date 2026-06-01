//go:build windows

package platform

import (
	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
	"github.com/yttydcs/myflowhub-clipboardnode/windows"
)

func newClipboardAdapter(_ string, opts ClipboardOptions) (clipboard.Adapter, error) {
	return windows.NewClipboardAdapter(windows.Options{MaxReadBytes: opts.MaxReadBytes})
}
