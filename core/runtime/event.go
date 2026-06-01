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
	ClipboardTextEventName     = "clipboard.text.v1"
	ClipboardTransferEventName = "clipboard.transfer.v1"
	EventVersionV1             = 1
	ContentTypeTextPlain       = "text/plain"
	EncodingUTF8               = "utf-8"
	maxEventEnvelopeBytes      = 4096
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

type TransferReference struct {
	Provider string `json:"provider"`
	OpaqueID string `json:"opaque_id,omitempty"`
	URI      string `json:"uri,omitempty"`
}

type ClipboardTransferManifestV1 struct {
	Version          int                 `json:"version"`
	EventID          string              `json:"event_id"`
	OriginNode       uint32              `json:"origin_node"`
	OriginInstanceID string              `json:"origin_instance_id"`
	OriginDevice     string              `json:"origin_device,omitempty"`
	ContentType      string              `json:"content_type"`
	Encoding         string              `json:"encoding"`
	Size             int                 `json:"size"`
	SHA256           string              `json:"sha256"`
	References       []TransferReference `json:"refs"`
	TS               int64               `json:"ts"`
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
	digest, err := InspectTextContent(text)
	if err != nil {
		return TextDigest{}, err
	}
	if digest.Size > maxInlineBytes {
		return TextDigest{}, fmt.Errorf("clipboard text size %d exceeds max_inline_bytes %d", digest.Size, maxInlineBytes)
	}
	return digest, nil
}

func InspectTextContent(text string) (TextDigest, error) {
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

func NewClipboardTransferManifestV1(digest TextDigest, refs []TransferReference, opts BuildEventOptions) (ClipboardTransferManifestV1, error) {
	if opts.OriginNode == 0 {
		return ClipboardTransferManifestV1{}, fmt.Errorf("origin_node is required")
	}
	opts.OriginInstanceID = strings.TrimSpace(opts.OriginInstanceID)
	if opts.OriginInstanceID == "" {
		return ClipboardTransferManifestV1{}, fmt.Errorf("origin_instance_id is required")
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
		return ClipboardTransferManifestV1{}, fmt.Errorf("create event id: %w", err)
	}
	manifest := ClipboardTransferManifestV1{
		Version:          EventVersionV1,
		EventID:          strings.TrimSpace(eventID),
		OriginNode:       opts.OriginNode,
		OriginInstanceID: opts.OriginInstanceID,
		OriginDevice:     strings.TrimSpace(opts.OriginDevice),
		ContentType:      ContentTypeTextPlain,
		Encoding:         EncodingUTF8,
		Size:             digest.Size,
		SHA256:           digest.SHA256,
		References:       refs,
		TS:               now().UnixMilli(),
	}
	if err := manifest.Validate(); err != nil {
		return ClipboardTransferManifestV1{}, err
	}
	return manifest, nil
}

func (m ClipboardTransferManifestV1) Validate() error {
	if m.Version != EventVersionV1 {
		return fmt.Errorf("unsupported transfer manifest version %d", m.Version)
	}
	if strings.TrimSpace(m.EventID) == "" {
		return fmt.Errorf("event_id is required")
	}
	if len(m.EventID) > 128 {
		return fmt.Errorf("event_id is too long")
	}
	if m.OriginNode == 0 {
		return fmt.Errorf("origin_node is required")
	}
	if strings.TrimSpace(m.OriginInstanceID) == "" {
		return fmt.Errorf("origin_instance_id is required")
	}
	if m.ContentType != ContentTypeTextPlain {
		return fmt.Errorf("unsupported content_type %q", m.ContentType)
	}
	if !strings.EqualFold(m.Encoding, EncodingUTF8) {
		return fmt.Errorf("unsupported encoding %q", m.Encoding)
	}
	if m.Size <= 0 {
		return fmt.Errorf("size must be positive")
	}
	if len(m.SHA256) != 64 {
		return fmt.Errorf("sha256 must be a hex encoded sha256 digest")
	}
	if _, err := hex.DecodeString(m.SHA256); err != nil {
		return fmt.Errorf("sha256 must be hex: %w", err)
	}
	if len(m.References) == 0 {
		return fmt.Errorf("at least one transfer reference is required")
	}
	for i, ref := range m.References {
		if strings.TrimSpace(ref.Provider) == "" {
			return fmt.Errorf("transfer reference %d provider is required", i)
		}
		if strings.TrimSpace(ref.OpaqueID) == "" && strings.TrimSpace(ref.URI) == "" {
			return fmt.Errorf("transfer reference %d requires opaque_id or uri", i)
		}
	}
	if m.TS <= 0 {
		return fmt.Errorf("ts is required")
	}
	return nil
}

func (m ClipboardTransferManifestV1) IsLocalOrigin(nodeID uint32, instanceID string) bool {
	if nodeID != 0 && m.OriginNode == nodeID {
		return true
	}
	if strings.TrimSpace(instanceID) != "" && m.OriginInstanceID == strings.TrimSpace(instanceID) {
		return true
	}
	return false
}

func MarshalClipboardTransferManifestV1(manifest ClipboardTransferManifestV1) ([]byte, error) {
	if err := manifest.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(manifest)
}

func ParseClipboardTransferManifestV1(payload []byte) (ClipboardTransferManifestV1, error) {
	if len(payload) == 0 {
		return ClipboardTransferManifestV1{}, fmt.Errorf("payload is required")
	}
	if len(payload) > maxEventEnvelopeBytes*2 {
		return ClipboardTransferManifestV1{}, fmt.Errorf("payload exceeds maximum transfer manifest size")
	}
	var manifest ClipboardTransferManifestV1
	if err := json.Unmarshal(payload, &manifest); err != nil {
		return ClipboardTransferManifestV1{}, fmt.Errorf("decode transfer manifest: %w", err)
	}
	if err := manifest.Validate(); err != nil {
		return ClipboardTransferManifestV1{}, err
	}
	return manifest, nil
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
