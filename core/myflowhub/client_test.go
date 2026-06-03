package myflowhub

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	prototopicbus "github.com/yttydcs/myflowhub-proto/protocol/topicbus"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"
)

func TestParseTopicBusPublish(t *testing.T) {
	body, err := sdktransport.EncodeMessage(prototopicbus.ActionPublish, prototopicbus.PublishReq{
		Topic:   "clipboard/dev",
		Name:    "clipboard.text.v1",
		TS:      1,
		Payload: json.RawMessage(`{"event_id":"evt-1"}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := parseTopicBusPublish(body)
	if err != nil {
		t.Fatalf("parseTopicBusPublish returned error: %v", err)
	}
	if req.Topic != "clipboard/dev" || req.Name != "clipboard.text.v1" {
		t.Fatalf("unexpected publish req: %+v", req)
	}
	if string(req.Payload) != `{"event_id":"evt-1"}` {
		t.Fatalf("payload = %s", req.Payload)
	}
}

func TestParseTopicBusPublishRejectsNonPublish(t *testing.T) {
	body, err := sdktransport.EncodeMessage(prototopicbus.ActionSubscribe, prototopicbus.SubscribeReq{Topic: "clipboard/dev"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseTopicBusPublish(body); err == nil {
		t.Fatalf("expected non-publish error")
	}
}

func TestAuthPayloadsIncludeDisplayName(t *testing.T) {
	reg := registerData(" local-device ", "pub-key")
	if reg.DeviceID != "local-device" {
		t.Fatalf("register device id = %q", reg.DeviceID)
	}
	if reg.DisplayName != "local-device" {
		t.Fatalf("register display name = %q", reg.DisplayName)
	}

	login := loginData(" local-device ", 14, 123, "nonce", "sig")
	if login.DeviceID != "local-device" {
		t.Fatalf("login device id = %q", login.DeviceID)
	}
	if login.DisplayName != "local-device" {
		t.Fatalf("login display name = %q", login.DisplayName)
	}
	if login.NodeID != 14 || login.TS != 123 || login.Nonce != "nonce" || login.Sig != "sig" || login.Alg != "ES256" {
		t.Fatalf("unexpected login payload: %+v", login)
	}
}

func TestLoadAuthSnapshotTreatsPersistedLoginAsStale(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auth_snapshot.json")
	data := []byte(`{
  "device_id": "local-device",
  "node_id": 14,
  "hub_id": 1,
  "logged_in": true,
  "last_action": "login_resp",
  "last_message": "ok"
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	client, err := New(Options{WorkDir: dir})
	if err != nil {
		t.Fatal(err)
	}
	st := client.AuthState()
	if st.LoggedIn {
		t.Fatalf("loaded persisted auth snapshot as logged in")
	}
	if st.DeviceID != "local-device" || st.NodeID != 14 || st.HubID != 1 {
		t.Fatalf("identity fields were not preserved: %+v", st)
	}
}

func TestClosePersistsLoggedOutSnapshot(t *testing.T) {
	client, err := New(Options{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	client.setAuthSnapshot(protoauth.ActionLoginResp, "local-device", protoauth.RespData{
		Code:   1,
		Msg:    "ok",
		NodeID: 14,
		HubID:  1,
	}, true)

	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(client.authSnapshotPath())
	if err != nil {
		t.Fatal(err)
	}
	var st AuthState
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	if st.LoggedIn {
		t.Fatalf("close persisted logged_in=true")
	}
	if st.DeviceID != "local-device" || st.NodeID != 14 || st.HubID != 1 {
		t.Fatalf("close dropped identity fields: %+v", st)
	}
}
