package bridge

import "encoding/json"

const (
	ActionConnect       = "connect"
	ActionSetConfig     = "set_config"
	ActionSendText      = "send_text"
	ActionReadClipboard = "read_clipboard"
	ActionApplyEvent    = "apply_event"
	ActionClearRecent   = "clear_recent"
	ActionShutdown      = "shutdown"

	EventStatusChanged   = "status.changed"
	EventActivityUpdated = "activity.updated"
	EventTransferUpdate  = "transfer.updated"
	EventClipboardRecv   = "clipboard.received"
	EventError           = "error"
)

type EngineCommand struct {
	ID     string          `json:"id"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data,omitempty"`
}

type EngineEvent struct {
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name"`
	Data  json.RawMessage `json:"data,omitempty"`
	OK    bool            `json:"ok"`
	Error string          `json:"error,omitempty"`
}

type Settings struct {
	Enabled          bool   `json:"enabled"`
	ParentEndpoint   string `json:"parent_endpoint"`
	Topic            string `json:"topic"`
	DeviceLabel      string `json:"device_label,omitempty"`
	MaxInlineBytes   int    `json:"max_inline_bytes"`
	AutoWatch        bool   `json:"auto_watch"`
	AutoApply        bool   `json:"auto_apply"`
	HistoryRetention string `json:"history_retention"`
	TransferProvider string `json:"transfer_provider,omitempty"`
	TransferRef      string `json:"transfer_ref,omitempty"`
}

type Status struct {
	Connected        bool   `json:"connected"`
	LoggedIn         bool   `json:"logged_in"`
	NodeID           uint32 `json:"node_id,omitempty"`
	HubID            uint32 `json:"hub_id,omitempty"`
	ParentEndpoint   string `json:"parent_endpoint"`
	Enabled          bool   `json:"enabled"`
	Topic            string `json:"topic"`
	DeviceLabel      string `json:"device_label,omitempty"`
	AutoWatch        bool   `json:"auto_watch"`
	AutoApply        bool   `json:"auto_apply"`
	TransferProvider string `json:"transfer_provider,omitempty"`
	TransferRef      string `json:"transfer_ref,omitempty"`
	Started          bool   `json:"started"`
	Subscribed       bool   `json:"subscribed"`
	Watching         bool   `json:"watching"`
	PendingEventID   string `json:"pending_event_id,omitempty"`
	PendingSize      int    `json:"pending_size,omitempty"`
	PendingHash      string `json:"pending_hash_prefix,omitempty"`
	LastAction       string `json:"last_action,omitempty"`
	LastEventID      string `json:"last_event_id,omitempty"`
	LastSize         int    `json:"last_size,omitempty"`
	LastHashPrefix   string `json:"last_hash_prefix,omitempty"`
	LastError        string `json:"last_error,omitempty"`
}

type Activity struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	DeviceLabel string `json:"device_label,omitempty"`
	ByteSize    int    `json:"byte_size"`
	HashPrefix  string `json:"hash_prefix,omitempty"`
	TimestampMS int64  `json:"timestamp_ms"`
}

type Transfer struct {
	ID         string `json:"id"`
	State      string `json:"state"`
	ByteSize   int    `json:"byte_size"`
	HashPrefix string `json:"hash_prefix,omitempty"`
	Detail     string `json:"detail,omitempty"`
}

type PlatformCapability struct {
	PlatformLabel  string   `json:"platform_label"`
	AutomaticWatch bool     `json:"automatic_watch"`
	ManualSend     bool     `json:"manual_send"`
	AutoApply      bool     `json:"auto_apply"`
	ShareSheet     bool     `json:"share_sheet"`
	Notes          []string `json:"notes,omitempty"`
}

func EncodeCommand(command EngineCommand) ([]byte, error) {
	return json.Marshal(command)
}

func DecodeCommand(payload []byte) (EngineCommand, error) {
	var command EngineCommand
	if err := json.Unmarshal(payload, &command); err != nil {
		return EngineCommand{}, err
	}
	return command, nil
}

func EncodeEvent(event EngineEvent) ([]byte, error) {
	return json.Marshal(event)
}

func DecodeEvent(payload []byte) (EngineEvent, error) {
	var event EngineEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return EngineEvent{}, err
	}
	return event, nil
}
