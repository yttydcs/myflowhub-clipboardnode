package myflowhub

import (
	"encoding/json"
	"testing"

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
