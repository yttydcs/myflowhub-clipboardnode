package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type nodeKeys struct {
	PrivKey string `json:"privkey"`
	PubKey  string `json:"pubkey"`
}

type KeyStore struct {
	mu   sync.Mutex
	path string

	priv *ecdsa.PrivateKey
	pub  string
}

func NewKeyStore(path string) *KeyStore {
	return &KeyStore{path: filepath.Clean(path)}
}

func (s *KeyStore) Ensure() (string, error) {
	if s == nil {
		return "", errors.New("keystore not initialized")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.priv != nil && strings.TrimSpace(s.pub) != "" {
		return s.pub, nil
	}
	priv, pub, err := loadOrCreateNodeKeys(s.path)
	if err != nil {
		return "", err
	}
	s.priv = priv
	s.pub = pub
	return pub, nil
}

func (s *KeyStore) SignLogin(deviceID string, nodeID uint32, ts int64, nonce string) (string, error) {
	if s == nil {
		return "", errors.New("keystore not initialized")
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return "", errors.New("device_id is required")
	}
	if nodeID == 0 {
		return "", errors.New("node_id is required")
	}
	nonce = strings.TrimSpace(nonce)
	if nonce == "" {
		return "", errors.New("nonce is required")
	}
	if _, err := s.Ensure(); err != nil {
		return "", err
	}
	s.mu.Lock()
	priv := s.priv
	s.mu.Unlock()
	if priv == nil {
		return "", errors.New("private key invalid")
	}
	return signLogin(priv, deviceID, nodeID, ts, nonce)
}

func loadOrCreateNodeKeys(path string) (*ecdsa.PrivateKey, string, error) {
	if priv, pub, err := readNodeKeys(path); err == nil && priv != nil && strings.TrimSpace(pub) != "" {
		return priv, pub, nil
	}
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, "", err
	}
	privDER, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, "", err
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, "", err
	}
	keys := nodeKeys{
		PrivKey: base64.StdEncoding.EncodeToString(privDER),
		PubKey:  base64.StdEncoding.EncodeToString(pubDER),
	}
	if err := writeNodeKeys(path, keys); err != nil {
		return nil, "", err
	}
	return priv, keys.PubKey, nil
}

func readNodeKeys(path string) (*ecdsa.PrivateKey, string, error) {
	path = filepath.Clean(path)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}
	var keys nodeKeys
	if err := json.Unmarshal(data, &keys); err != nil {
		return nil, "", err
	}
	priv, err := parsePrivateKey(keys.PrivKey)
	if err != nil {
		return nil, "", err
	}
	if strings.TrimSpace(keys.PubKey) == "" {
		return nil, "", errors.New("pubkey is empty")
	}
	return priv, strings.TrimSpace(keys.PubKey), nil
}

func writeNodeKeys(path string, keys nodeKeys) error {
	path = filepath.Clean(path)
	if strings.TrimSpace(path) == "" || path == "." {
		return errors.New("keystore path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create keystore directory: %w", err)
	}
	data, err := json.MarshalIndent(keys, "", "  ")
	if err != nil {
		return fmt.Errorf("encode node keys: %w", err)
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func parsePrivateKey(encoded string) (*ecdsa.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
	if err != nil {
		return nil, err
	}
	priv, err := x509.ParseECPrivateKey(raw)
	if err != nil {
		return nil, err
	}
	if priv == nil || priv.Curve != elliptic.P256() {
		return nil, errors.New("private key is not p256")
	}
	return priv, nil
}

func GenerateNonce(n int) string {
	if n <= 0 {
		n = 12
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return ""
	}
	return hex.EncodeToString(buf)
}

func signLogin(priv *ecdsa.PrivateKey, deviceID string, nodeID uint32, ts int64, nonce string) (string, error) {
	if priv == nil {
		return "", errors.New("private key is required")
	}
	sum := sha256.Sum256(loginSignBytes(deviceID, nodeID, ts, nonce))
	sig, err := ecdsa.SignASN1(rand.Reader, priv, sum[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

func loginSignBytes(deviceID string, nodeID uint32, ts int64, nonce string) []byte {
	var b strings.Builder
	b.WriteString("login\n")
	b.WriteString(strings.TrimSpace(deviceID))
	b.WriteByte('\n')
	b.WriteString(strconv.FormatUint(uint64(nodeID), 10))
	b.WriteByte('\n')
	b.WriteString(strconv.FormatInt(ts, 10))
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(nonce))
	return []byte(b.String())
}
