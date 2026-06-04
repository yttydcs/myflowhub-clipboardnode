package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yttydcs/myflowhub-clipboardnode/core/configstore"
	"github.com/yttydcs/myflowhub-clipboardnode/core/engine"
	"github.com/yttydcs/myflowhub-clipboardnode/platform"
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
	sendText := flag.String("send-text", "", "publish one manual text value through the configured clipboard topic")
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

	workDir := filepath.Dir(*configFile)
	eng, err := engine.New(engine.Options{
		Config:    cfg,
		WorkDir:   workDir,
		Clipboard: adapter,
		Log:       slog.Default(),
	})
	if err != nil {
		return fmt.Errorf("initialize clipboard engine: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	fmt.Printf("ClipboardNode config loaded from %s\n", *configFile)
	fmt.Printf("parent_endpoint=%q topic=%q topics=%d enabled=%t auto_watch=%t auto_apply=%t max_inline_bytes=%d\n",
		cfg.ParentEndpoint, cfg.Topic, len(cfg.Topics), cfg.Enabled, cfg.AutoWatch, cfg.AutoApply, cfg.MaxInlineBytes)
	if err := eng.Start(ctx); err != nil {
		return err
	}
	defer func() {
		_ = eng.Stop(context.Background())
	}()
	if strings.TrimSpace(*sendText) != "" {
		decision, err := eng.SendText(ctx, *sendText)
		if err != nil {
			return err
		}
		fmt.Printf("manual send action=%s topic=%q event_id=%s size=%d hash=%s\n",
			decision.Action, decision.Topic, decision.EventID, decision.Size, decision.HashPrefix)
		return nil
	}
	fmt.Println("ClipboardNode running; press Ctrl+C to stop.")
	<-ctx.Done()
	status := eng.Status()
	fmt.Printf("stopped; connected=%t logged_in=%t node=%d hub=%d last_action=%s last_error=%q\n",
		status.Connected, status.LoggedIn, status.NodeID, status.HubID, status.Runtime.LastAction, status.LastError)
	return nil
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config directory: %w", err)
	}
	return filepath.Join(dir, "MyFlowHub", "ClipboardNode", "config.json"), nil
}
