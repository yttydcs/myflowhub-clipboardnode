package auth

import (
	"path/filepath"
	"testing"
)

func TestKeyStoreEnsurePersistsAndReusesKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "node_keys.json")
	store := NewKeyStore(path)
	pub1, err := store.Ensure()
	if err != nil {
		t.Fatalf("Ensure returned error: %v", err)
	}
	if pub1 == "" {
		t.Fatal("pubkey is empty")
	}

	store2 := NewKeyStore(path)
	pub2, err := store2.Ensure()
	if err != nil {
		t.Fatalf("Ensure reload returned error: %v", err)
	}
	if pub2 != pub1 {
		t.Fatalf("pubkey changed after reload")
	}
}

func TestKeyStoreSignLoginValidatesInput(t *testing.T) {
	store := NewKeyStore(filepath.Join(t.TempDir(), "node_keys.json"))
	if _, err := store.SignLogin("", 12, 1, "nonce"); err == nil {
		t.Fatalf("expected empty device_id error")
	}
	if _, err := store.SignLogin("device", 0, 1, "nonce"); err == nil {
		t.Fatalf("expected node_id error")
	}
	if _, err := store.SignLogin("device", 12, 1, ""); err == nil {
		t.Fatalf("expected nonce error")
	}
	sig, err := store.SignLogin("device", 12, 1, "nonce")
	if err != nil {
		t.Fatalf("SignLogin returned error: %v", err)
	}
	if sig == "" {
		t.Fatal("signature is empty")
	}
}
