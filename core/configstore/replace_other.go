//go:build !windows

package configstore

import "os"

func replaceFile(source string, destination string) error {
	return os.Rename(source, destination)
}
