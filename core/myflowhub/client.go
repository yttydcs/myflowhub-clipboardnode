package myflowhub

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	core "github.com/yttydcs/myflowhub-core"
	"github.com/yttydcs/myflowhub-core/header"
	protoauth "github.com/yttydcs/myflowhub-proto/protocol/auth"
	prototopicbus "github.com/yttydcs/myflowhub-proto/protocol/topicbus"
	sdkawait "github.com/yttydcs/myflowhub-sdk/await"
	"github.com/yttydcs/myflowhub-sdk/session"
	sdktransport "github.com/yttydcs/myflowhub-sdk/transport"

	"github.com/yttydcs/myflowhub-clipboardnode/core/auth"
	coreruntime "github.com/yttydcs/myflowhub-clipboardnode/core/runtime"
)

const defaultRequestTimeout = 8 * time.Second

type AuthState struct {
	DeviceID     string `json:"device_id,omitempty"`
	NodeID       uint32 `json:"node_id,omitempty"`
	HubID        uint32 `json:"hub_id,omitempty"`
	Role         string `json:"role,omitempty"`
	LoggedIn     bool   `json:"logged_in"`
	LastAction   string `json:"last_action,omitempty"`
	LastMessage  string `json:"last_message,omitempty"`
	LastUnixTime int64  `json:"last_unix_time,omitempty"`
}

type Status struct {
	Connected bool
	Auth      AuthState
	LastError string
}

type Options struct {
	WorkDir        string
	Log            *slog.Logger
	RequestTimeout time.Duration
	EventBuffer    int
}

type Client struct {
	log     *slog.Logger
	workDir string
	timeout time.Duration

	clientMu sync.Mutex
	client   *sdkawait.Client
	endpoint string

	connected atomic.Bool
	lastErr   atomic.Value // string

	keys *auth.KeyStore

	authMu sync.Mutex
	auth   AuthState

	events chan coreruntime.TopicBusMessage
}

func New(opts Options) (*Client, error) {
	workDir := strings.TrimSpace(opts.WorkDir)
	if workDir == "" {
		return nil, errors.New("workDir is required")
	}
	abs := workDir
	if !filepath.IsAbs(abs) {
		if cwd, err := os.Getwd(); err == nil {
			abs = filepath.Join(cwd, abs)
		}
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return nil, fmt.Errorf("create workDir: %w", err)
	}
	timeout := opts.RequestTimeout
	if timeout <= 0 {
		timeout = defaultRequestTimeout
	}
	buffer := opts.EventBuffer
	if buffer <= 0 {
		buffer = 64
	}
	log := opts.Log
	if log == nil {
		log = slog.Default()
	}
	c := &Client{
		log:     log,
		workDir: abs,
		timeout: timeout,
		keys:    auth.NewKeyStore(filepath.Join(abs, "node_keys.json")),
		events:  make(chan coreruntime.TopicBusMessage, buffer),
	}
	_ = c.loadAuthSnapshot()
	return c, nil
}

func (c *Client) Connect(ctx context.Context, endpoint string) error {
	if c == nil {
		return errors.New("client is nil")
	}
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return errors.New("endpoint is required")
	}
	c.clientMu.Lock()
	prevEndpoint := c.endpoint
	c.clientMu.Unlock()
	if prevEndpoint != "" && prevEndpoint != endpoint {
		_ = c.Close()
	}
	sdk := c.ensureClient()
	errCh := make(chan error, 1)
	go func() {
		errCh <- sdk.Connect(endpoint)
	}()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, session.ErrAlreadyConnected) {
			c.storeLastError(err)
			return err
		}
	case <-ctx.Done():
		sdk.Close()
		err := fmt.Errorf("connect canceled: %w", ctx.Err())
		c.storeLastError(err)
		return err
	}
	c.connected.Store(true)
	c.clientMu.Lock()
	c.endpoint = endpoint
	c.clientMu.Unlock()
	return nil
}

func (c *Client) EnsureIdentity(ctx context.Context, deviceID string) (AuthState, error) {
	if c == nil {
		return AuthState{}, errors.New("client is nil")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return AuthState{}, errors.New("device_id is required")
	}
	st := c.AuthState()
	if st.NodeID != 0 {
		if !st.LoggedIn || strings.TrimSpace(st.DeviceID) != deviceID {
			return c.Login(ctx, deviceID, st.NodeID)
		}
		return st, nil
	}
	if _, err := c.Register(ctx, deviceID); err != nil {
		return AuthState{}, err
	}
	st = c.AuthState()
	if st.NodeID == 0 {
		return AuthState{}, errors.New("register did not return node_id")
	}
	return c.Login(ctx, deviceID, st.NodeID)
}

func (c *Client) Register(ctx context.Context, deviceID string) (AuthState, error) {
	if c == nil {
		return AuthState{}, errors.New("client is nil")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return AuthState{}, errors.New("device_id is required")
	}
	if !c.IsConnected() {
		return AuthState{}, errors.New("not connected")
	}
	pub, err := c.keys.Ensure()
	if err != nil {
		c.storeLastError(err)
		return AuthState{}, err
	}
	payload, err := sdktransport.EncodeMessage(protoauth.ActionRegister, registerData(deviceID, pub))
	if err != nil {
		return AuthState{}, err
	}
	resp, err := c.sendAndAwait(ctx, protoauth.SubProtoAuth, 0, 0, payload, protoauth.ActionRegisterResp)
	if err != nil {
		c.setAuthResult(false, protoauth.ActionRegisterResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	var out protoauth.RespData
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		c.setAuthResult(false, protoauth.ActionRegisterResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	if out.Code != 1 {
		err := responseError("auth register", out.Code, out.Msg)
		c.setAuthResult(false, protoauth.ActionRegisterResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	c.setAuthSnapshot(protoauth.ActionRegisterResp, deviceID, out, true)
	_ = c.saveAuthSnapshot(c.AuthState())
	return c.AuthState(), nil
}

func (c *Client) Login(ctx context.Context, deviceID string, nodeID uint32) (AuthState, error) {
	if c == nil {
		return AuthState{}, errors.New("client is nil")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return AuthState{}, errors.New("device_id is required")
	}
	if nodeID == 0 {
		return AuthState{}, errors.New("node_id is required")
	}
	if !c.IsConnected() {
		return AuthState{}, errors.New("not connected")
	}
	ts := time.Now().Unix()
	nonce := auth.GenerateNonce(12)
	if nonce == "" {
		return AuthState{}, errors.New("create login nonce failed")
	}
	sig, err := c.keys.SignLogin(deviceID, nodeID, ts, nonce)
	if err != nil {
		c.storeLastError(err)
		return AuthState{}, err
	}
	payload, err := sdktransport.EncodeMessage(protoauth.ActionLogin, loginData(deviceID, nodeID, ts, nonce, sig))
	if err != nil {
		return AuthState{}, err
	}
	resp, err := c.sendAndAwait(ctx, protoauth.SubProtoAuth, 0, 0, payload, protoauth.ActionLoginResp)
	if err != nil {
		c.setAuthResult(false, protoauth.ActionLoginResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	var out protoauth.RespData
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		c.setAuthResult(false, protoauth.ActionLoginResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	if out.Code != 1 {
		err := responseError("auth login", out.Code, out.Msg)
		c.setAuthResult(false, protoauth.ActionLoginResp, err.Error())
		c.storeLastError(err)
		return AuthState{}, err
	}
	c.setAuthSnapshot(protoauth.ActionLoginResp, deviceID, out, true)
	_ = c.saveAuthSnapshot(c.AuthState())
	return c.AuthState(), nil
}

func (c *Client) Subscribe(ctx context.Context, topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("topic is required")
	}
	auth := c.AuthState()
	if !auth.LoggedIn || auth.NodeID == 0 || auth.HubID == 0 {
		return errors.New("login required")
	}
	payload, err := sdktransport.EncodeMessage(prototopicbus.ActionSubscribe, prototopicbus.SubscribeReq{Topic: topic})
	if err != nil {
		return err
	}
	resp, err := c.sendAndAwait(ctx, prototopicbus.SubProtoTopicBus, auth.NodeID, auth.HubID, payload, prototopicbus.ActionSubscribeResp)
	if err != nil {
		c.storeLastError(err)
		return err
	}
	var out prototopicbus.Resp
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		c.storeLastError(err)
		return err
	}
	if out.Code != 1 {
		err := responseError("topicbus subscribe", out.Code, out.Msg)
		c.storeLastError(err)
		return err
	}
	return nil
}

func (c *Client) Unsubscribe(ctx context.Context, topic string) error {
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return nil
	}
	auth := c.AuthState()
	if !auth.LoggedIn || auth.NodeID == 0 || auth.HubID == 0 || !c.IsConnected() {
		return nil
	}
	payload, err := sdktransport.EncodeMessage(prototopicbus.ActionUnsubscribe, prototopicbus.SubscribeReq{Topic: topic})
	if err != nil {
		return err
	}
	resp, err := c.sendAndAwait(ctx, prototopicbus.SubProtoTopicBus, auth.NodeID, auth.HubID, payload, prototopicbus.ActionUnsubscribeResp)
	if err != nil {
		c.storeLastError(err)
		return err
	}
	var out prototopicbus.Resp
	if err := json.Unmarshal(resp.Message.Data, &out); err != nil {
		c.storeLastError(err)
		return err
	}
	if out.Code != 1 {
		err := responseError("topicbus unsubscribe", out.Code, out.Msg)
		c.storeLastError(err)
		return err
	}
	return nil
}

func (c *Client) Publish(ctx context.Context, topic string, name string, payload []byte) error {
	topic = strings.TrimSpace(topic)
	name = strings.TrimSpace(name)
	if topic == "" {
		return errors.New("topic is required")
	}
	if name == "" {
		return errors.New("name is required")
	}
	auth := c.AuthState()
	if !auth.LoggedIn || auth.NodeID == 0 || auth.HubID == 0 {
		return errors.New("login required")
	}
	reqPayload, err := sdktransport.EncodeMessage(prototopicbus.ActionPublish, prototopicbus.PublishReq{
		Topic:   topic,
		Name:    name,
		TS:      time.Now().UnixMilli(),
		Payload: json.RawMessage(append([]byte(nil), payload...)),
	})
	if err != nil {
		return err
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(prototopicbus.SubProtoTopicBus).
		WithSourceID(auth.NodeID).
		WithTargetID(auth.HubID).
		WithTimestamp(uint32(time.Now().Unix()))
	errCh := make(chan error, 1)
	go func() {
		errCh <- c.ensureClient().Send(hdr, reqPayload)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			c.storeLastError(err)
		}
		return err
	case <-ctx.Done():
		err := ctx.Err()
		c.storeLastError(err)
		return err
	}
}

func (c *Client) Events() <-chan coreruntime.TopicBusMessage {
	if c == nil {
		return nil
	}
	return c.events
}

func (c *Client) AuthState() AuthState {
	if c == nil {
		return AuthState{}
	}
	c.authMu.Lock()
	st := c.auth
	c.authMu.Unlock()
	return st
}

func (c *Client) Status() Status {
	if c == nil {
		return Status{}
	}
	lastErr := ""
	if v := c.lastErr.Load(); v != nil {
		if s, ok := v.(string); ok {
			lastErr = s
		}
	}
	return Status{
		Connected: c.IsConnected(),
		Auth:      c.AuthState(),
		LastError: lastErr,
	}
}

func (c *Client) IsConnected() bool {
	return c != nil && c.connected.Load()
}

func (c *Client) ClearAuth() error {
	if c == nil {
		return errors.New("client is nil")
	}
	c.authMu.Lock()
	c.auth = AuthState{}
	c.authMu.Unlock()
	return c.saveAuthSnapshot(AuthState{})
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.clientMu.Lock()
	sdk := c.client
	c.client = nil
	c.endpoint = ""
	c.clientMu.Unlock()
	if sdk != nil {
		sdk.Close()
	}
	c.connected.Store(false)
	c.authMu.Lock()
	st := c.auth
	st.LoggedIn = false
	c.auth = st
	c.authMu.Unlock()
	if err := c.saveAuthSnapshot(st); err != nil {
		c.storeLastError(err)
		return err
	}
	return nil
}

func (c *Client) ensureClient() *sdkawait.Client {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	if c.client != nil {
		return c.client
	}
	c.client = sdkawait.NewClient(context.Background(), c.onUnmatchedFrame, c.onClientError)
	return c.client
}

func (c *Client) sendAndAwait(ctx context.Context, sub uint8, src, tgt uint32, payload []byte, expectAction string) (sdkawait.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}
	hdr := (&header.HeaderTcp{}).
		WithMajor(header.MajorCmd).
		WithSubProto(sub).
		WithSourceID(src).
		WithTargetID(tgt).
		WithTimestamp(uint32(time.Now().Unix()))
	return c.ensureClient().SendAndAwait(ctx, hdr, payload, expectAction)
}

func (c *Client) onClientError(err error) {
	if err == nil {
		return
	}
	c.connected.Store(false)
	c.storeLastError(err)
	if c.log != nil {
		c.log.Warn("myflowhub session error", "err", err.Error())
	}
}

func (c *Client) onUnmatchedFrame(hdr core.IHeader, payload []byte) {
	if c == nil || hdr == nil || hdr.SubProto() != prototopicbus.SubProtoTopicBus || len(payload) == 0 {
		return
	}
	switch hdr.Major() {
	case header.MajorCmd, header.MajorMsg:
	default:
		return
	}
	req, err := parseTopicBusPublish(payload)
	if err != nil {
		if c.log != nil {
			c.log.Warn("topicbus publish decode failed", "err", err.Error())
		}
		return
	}
	msg := coreruntime.TopicBusMessage{
		Topic:   req.Topic,
		Name:    req.Name,
		Payload: append([]byte(nil), req.Payload...),
	}
	select {
	case c.events <- msg:
	default:
		c.storeLastError(errors.New("topicbus event queue full"))
		if c.log != nil {
			c.log.Warn("topicbus event queue full", "topic", req.Topic, "name", req.Name)
		}
	}
}

func parseTopicBusPublish(payload []byte) (prototopicbus.PublishReq, error) {
	msg, err := sdktransport.DecodeMessage(payload)
	if err != nil {
		return prototopicbus.PublishReq{}, err
	}
	if msg.Action != prototopicbus.ActionPublish {
		return prototopicbus.PublishReq{}, fmt.Errorf("unexpected topicbus action %q", msg.Action)
	}
	var req prototopicbus.PublishReq
	if err := json.Unmarshal(msg.Data, &req); err != nil {
		return prototopicbus.PublishReq{}, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return prototopicbus.PublishReq{}, errors.New("publish name is required")
	}
	return req, nil
}

func (c *Client) setAuthSnapshot(action string, deviceID string, data protoauth.RespData, loggedIn bool) {
	c.authMu.Lock()
	c.auth = AuthState{
		DeviceID:     strings.TrimSpace(deviceID),
		NodeID:       data.NodeID,
		HubID:        data.HubID,
		Role:         strings.TrimSpace(data.Role),
		LoggedIn:     loggedIn,
		LastAction:   strings.TrimSpace(action),
		LastMessage:  strings.TrimSpace(data.Msg),
		LastUnixTime: time.Now().Unix(),
	}
	c.authMu.Unlock()
}

func (c *Client) setAuthResult(ok bool, action string, msg string) {
	c.authMu.Lock()
	st := c.auth
	st.LoggedIn = ok
	st.LastAction = strings.TrimSpace(action)
	st.LastMessage = strings.TrimSpace(msg)
	st.LastUnixTime = time.Now().Unix()
	c.auth = st
	c.authMu.Unlock()
}

func (c *Client) storeLastError(err error) {
	if c == nil {
		return
	}
	if err == nil {
		c.lastErr.Store("")
		return
	}
	c.lastErr.Store(err.Error())
}

func (c *Client) authSnapshotPath() string {
	if c == nil {
		return ""
	}
	return filepath.Join(c.workDir, "auth_snapshot.json")
}

func (c *Client) loadAuthSnapshot() error {
	path := c.authSnapshotPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	var st AuthState
	if err := json.Unmarshal(data, &st); err != nil {
		return err
	}
	st.LoggedIn = false
	c.authMu.Lock()
	c.auth = st
	c.authMu.Unlock()
	return nil
}

func (c *Client) saveAuthSnapshot(st AuthState) error {
	path := c.authSnapshotPath()
	if strings.TrimSpace(path) == "" {
		return errors.New("auth snapshot path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create auth snapshot directory: %w", err)
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return fmt.Errorf("encode auth snapshot: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func responseError(prefix string, code int, msg string) error {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		msg = fmt.Sprintf("%s failed (code=%d)", prefix, code)
	}
	return errors.New(msg)
}

func registerData(deviceID string, pub string) protoauth.RegisterData {
	deviceID = strings.TrimSpace(deviceID)
	return protoauth.RegisterData{
		DeviceID:    deviceID,
		PubKey:      pub,
		NodePub:     pub,
		DisplayName: deviceID,
	}
}

func loginData(deviceID string, nodeID uint32, ts int64, nonce string, sig string) protoauth.LoginData {
	deviceID = strings.TrimSpace(deviceID)
	return protoauth.LoginData{
		DeviceID:    deviceID,
		NodeID:      nodeID,
		DisplayName: deviceID,
		TS:          ts,
		Nonce:       nonce,
		Sig:         sig,
		Alg:         "ES256",
	}
}
