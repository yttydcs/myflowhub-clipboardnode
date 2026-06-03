package engine

import (
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yttydcs/myflowhub-clipboardnode/core/myflowhub"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

func TestStartClosesTransportAfterAuthFailure(t *testing.T) {
	endpoint := startBlackholeTCPServer(t)
	transport, err := myflowhub.New(myflowhub.Options{
		WorkDir:        t.TempDir(),
		RequestTimeout: 50 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	eng, err := New(Options{
		Config: coreruntime.Config{
			ParentEndpoint: endpoint,
			DeviceLabel:    "local-device",
		},
		WorkDir:   t.TempDir(),
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = eng.Start(ctx)
	if err == nil {
		t.Fatalf("expected auth failure")
	}
	if !strings.Contains(err.Error(), "authenticate myflowhub node") {
		t.Fatalf("unexpected start error: %v", err)
	}
	status := transport.Status()
	if status.Connected {
		t.Fatalf("transport remained connected after startup failure: %+v", status)
	}
	if status.Auth.LoggedIn {
		t.Fatalf("transport remained logged in after startup failure: %+v", status.Auth)
	}
}

func startBlackholeTCPServer(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	conns := make([]net.Conn, 0, 1)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, conn)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		_ = ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range conns {
			_ = conn.Close()
		}
	})
	return ln.Addr().String()
}
