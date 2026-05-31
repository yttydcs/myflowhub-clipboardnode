package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	ClipboardTextEventName = "clipboard.text.v1"
	EventVersionV1         = 1
	ContentTypeTextPlain   = "text/plain"
	EncodingUTF8           = "utf-8"
	maxEventEnvelopeBytes  = 4096
)

type ClipboardTextEventV1 struct {
	Version          int    `json:"version"`
	EventID          string `json:"event_id"`
	OriginNode       uint32 `json:"origin_node"`
	OriginInstanceID string `json:"origin_instance_id"`
	OriginDevice     string `json:"origin_device,omitempty"`
	ContentType      string `json:"content_type"`
	Encoding         string `json:"encoding"`
	Size             int    `json:"size"`
	SHA256           string `json:"sha256"`
	Text             string `json:"text"`
	TS               int64  `json:"ts"`
}

type TextDigest struct {
	Size   int
	SHA256 string
}

type BuildEventOptions struct {
	OriginNode       uint32
	OriginInstanceID string
	OriginDevice     string
	MaxInlineBytes   int
	Now              func() time.Time
	NewEventID       func() (string, error)
}

func InspectText(text string, maxInlineBytes int) (TextDigest, error) {
	if maxInlineBytes <= 0 {
		return TextDigest{}, fmt.Errorf("max_inline_bytes must be positive")
	}
	if text == "" {
		return TextDigest{}, fmt.Errorf("clipboard text is empty")
	}
	if !utf8.ValidString(text) {
		return TextDigest{}, fmt.Errorf("clipboard text is not valid utf-8")
	}
	if strings.ContainsRune(text, '\x00') {
		return TextDigest{}, fmt.Errorf("clipboard text contains a null character")
	}
	size := len(text)
	if size > maxInlineBytes {
		return TextDigest{}, fmt.Errorf("clipboard text size %d exceeds max_inline_bytes %d", size, maxInlineBytes)
	}
	return TextDigest{Size: size, SHA256: HashText(text)}, nil
}

func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

func NewClipboardTextEventV1(text string, opts BuildEventOptions) (ClipboardTextEventV1, error) {
	if opts.OriginNode == 0 {
		return ClipboardTextEventV1{}, fmt.Errorf("origin_node is required")
	}
	opts.OriginInstanceID = strings.TrimSpace(opts.OriginInstanceID)
	if opts.OriginInstanceID == "" {
		return ClipboardTextEventV1{}, fmt.Errorf("origin_instance_id is required")
	}
	digest, err := InspectText(text, opts.MaxInlineBytes)
	if err != nil {
		return ClipboardTextEventV1{}, err
	}
	return newClipboardTextEventV1WithDigest(text, digest, opts)
}

func newClipboardTextEventV1WithDigest(text string, digest TextDigest, opts BuildEventOptions) (ClipboardTextEventV1, error) {
	if opts.OriginNode == 0 {
		return ClipboardTextEventV1{}, fmt.Errorf("origin_node is required")
	}
	opts.OriginInstanceID = strings.TrimSpace(opts.OriginInstanceID)
	if opts.OriginInstanceID == "" {
		return ClipboardTextEventV1{}, fmt.Errorf("origin_instance_id is required")
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	newEventID := opts.NewEventID
	if newEventID == nil {
		newEventID = RandomEventID
	}
	eventID, err := newEventID()
	if err != nil {
		return ClipboardTextEventV1{}, fmt.Errorf("create event id: %w", err)
	}
	evt := ClipboardTextEventV1{
		Version:          EventVersionV1,
		EventID:          strings.TrimSpace(eventID),
		OriginNode:       opts.OriginNode,
		OriginInstanceID: opts.OriginInstanceID,
		OriginDevice:     strings.TrimSpace(opts.OriginDevice),
		ContentType:      ContentTypeTextPlain,
		Encoding:         EncodingUTF8,
		Size:             digest.Size,
		SHA256:           digest.SHA256,
		Text:             text,
		TS:               now().UnixMilli(),
	}
	if err := evt.Validate(opts.MaxInlineBytes); err != nil {
		return ClipboardTextEventV1{}, err
	}
	return evt, nil
}

func (e ClipboardTextEventV1) Validate(maxInlineBytes int) error {
	if maxInlineBytes <= 0 {
		return fmt.Errorf("max_inline_bytes must be positive")
	}
	if e.Version != EventVersionV1 {
		return fmt.Errorf("unsupported clipboard event version %d", e.Version)
	}
	if strings.TrimSpace(e.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if len(e.EventID) > 128 {
		return fmt.Errorf("event_id is too long")
	}
	if e.OriginNode == 0 {
		return fmt.Errorf("origin_node is required")
	}
	if strings.TrimSpace(e.OriginInstanceID) == "" {
		return fmt.Errorf("origin_instance_id is required")
	}
	if e.ContentType != ContentTypeTextPlain {
		return fmt.Errorf("unsupported content_type %q", e.ContentType)
	}
	if !strings.EqualFold(e.Encoding, EncodingUTF8) {
		return fmt.Errorf("unsupported encoding %q", e.Encoding)
	}
	digest, err := InspectText(e.Text, maxInlineBytes)
	if err != nil {
		return err
	}
	if e.Size != digest.Size {
		return fmt.Errorf("size mismatch: got %d want %d", e.Size, digest.Size)
	}
	if e.SHA256 != digest.SHA256 {
		return fmt.Errorf("sha256 mismatch")
	}
	if e.TS <= 0 {
		return fmt.Errorf("ts is required")
	}
	return nil
}

func (e ClipboardTextEventV1) IsLocalOrigin(nodeID uint32, instanceID string) bool {
	if nodeID != 0 && e.OriginNode == nodeID {
		return true
	}
	if strings.TrimSpace(instanceID) != "" && e.OriginInstanceID == strings.TrimSpace(instanceID) {
		return true
	}
	return false
}

func MarshalClipboardTextEventV1(evt ClipboardTextEventV1, maxInlineBytes int) ([]byte, error) {
	if err := evt.Validate(maxInlineBytes); err != nil {
		return nil, err
	}
	return json.Marshal(evt)
}

func ParseClipboardTextEventV1(payload []byte, maxInlineBytes int) (ClipboardTextEventV1, error) {
	if len(payload) == 0 {
		return ClipboardTextEventV1{}, fmt.Errorf("payload is required")
	}
	if len(payload) > maxInlineBytes+maxEventEnvelopeBytes {
		return ClipboardTextEventV1{}, fmt.Errorf("payload exceeds maximum event envelope size")
	}
	var evt ClipboardTextEventV1
	if err := json.Unmarshal(payload, &evt); err != nil {
		return ClipboardTextEventV1{}, fmt.Errorf("decode clipboard event: %w", err)
	}
	if err := evt.Validate(maxInlineBytes); err != nil {
		return ClipboardTextEventV1{}, err
	}
	return evt, nil
}

func RandomEventID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}
