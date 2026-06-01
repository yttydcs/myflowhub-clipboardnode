package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/yttydcs/myflowhub-clipboardnode/bridge"
)

func TestValidateLoopbackListen(t *testing.T) {
	for _, address := range []string{"127.0.0.1:18291", "localhost:18291", "[::1]:18291"} {
		if err := validateLoopbackListen(address); err != nil {
			t.Fatalf("expected %s to be valid: %v", address, err)
		}
	}
}

func TestValidateLoopbackListenRejectsNonLoopback(t *testing.T) {
	for _, address := range []string{"0.0.0.0:18291", "192.168.1.10:18291", ":18291"} {
		if err := validateLoopbackListen(address); err == nil {
			t.Fatalf("expected %s to be rejected", address)
		}
	}
}

func TestHandleReturnsCommandErrors(t *testing.T) {
	var out bytes.Buffer
	host := &stdioHost{out: &out}
	_, err := host.handle(context.Background(), bridge.EngineCommand{
		ID:     "cmd-1",
		Action: "unknown",
	})
	if err == nil {
		t.Fatal("expected unsupported command error")
	}
	if !strings.Contains(out.String(), `"ok":false`) {
		t.Fatalf("expected error event, got %s", out.String())
	}
}
