package platform

import (
	"runtime"

	"github.com/yttydcs/myflowhub-clipboardnode/core/clipboard"
)

type ClipboardOptions struct {
	MaxReadBytes int
}

func NewClipboardAdapter(opts ClipboardOptions) (clipboard.Adapter, error) {
	return newClipboardAdapter(runtime.GOOS, opts)
}
