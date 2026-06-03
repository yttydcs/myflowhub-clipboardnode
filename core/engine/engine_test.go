package engine

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
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

func TestUpdateConfigClearsAuthWhenDeviceIDChanges(t *testing.T) {
	authDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(authDir, "auth_snapshot.json"), []byte(`{
  "device_id": "old-device",
  "node_id": 14,
  "hub_id": 1,
  "logged_in": true
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	transport, err := myflowhub.New(myflowhub.Options{WorkDir: authDir})
	if err != nil {
		t.Fatal(err)
	}
	eng, err := New(Options{
		Config: coreruntime.Config{
			DeviceID:    "old-device",
			DisplayName: "Old Device",
		},
		WorkDir:   t.TempDir(),
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := eng.UpdateConfig(context.Background(), coreruntime.Config{
		DeviceID:    "new-device",
		DisplayName: "New Device",
	}); err != nil {
		t.Fatal(err)
	}
	st := transport.AuthState()
	if st.DeviceID != "" || st.NodeID != 0 || st.HubID != 0 || st.LoggedIn {
		t.Fatalf("auth state was not cleared: %+v", st)
	}
	data, err := os.ReadFile(filepath.Join(authDir, "auth_snapshot.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted myflowhub.AuthState
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if persisted.DeviceID != "" || persisted.NodeID != 0 || persisted.HubID != 0 || persisted.LoggedIn {
		t.Fatalf("persisted auth state was not cleared: %+v", persisted)
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
