package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yttydcs/myflowhub-clipboardnode/core/configstore"
	"github.com/yttydcs/myflowhub-clipboardnode/windows"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	defaultConfigPath, err := configPath()
	if err != nil {
		return err
	}
	configFile := flag.String("config", defaultConfigPath, "path to ClipboardNode JSON config")
	flag.Parse()

	store, err := configstore.New(*configFile)
	if err != nil {
		return err
	}
	cfg, err := store.Load()
	if err != nil {
		return err
	}
	adapter, err := windows.NewClipboardAdapter(windows.Options{MaxReadBytes: cfg.MaxInlineBytes})
	if err != nil {
		return fmt.Errorf("initialize clipboard adapter: %w", err)
	}
	defer adapter.Close()

	if cfg.Enabled {
		return fmt.Errorf("clipboard sync is enabled, but the TopicBus SDK transport is not wired into this host skeleton yet")
	}
	fmt.Printf("ClipboardNode config loaded from %s\n", *configFile)
	fmt.Printf("sync disabled; topic=%q max_inline_bytes=%d\n", cfg.Topic, cfg.MaxInlineBytes)
	return nil
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "MyFlowHub", "ClipboardNode", "config.json"), nil
}
