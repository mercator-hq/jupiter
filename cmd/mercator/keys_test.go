package main

import (
	"crypto/ed25519"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeys(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "keys-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set flags
	keysFlags.output = tmpDir
	keysFlags.keyID = "test-key"
	keysFlags.format = "pem"
	keysFlags.noConfirm = true

	// Generate keys
	err = generateKeys(nil, []string{})
	if err != nil {
		t.Fatalf("generateKeys() error = %v", err)
	}

	// Verify public key file exists
	publicKeyPath := filepath.Join(tmpDir, "test-key_public.pem")
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file was not created")
	}

	// Verify private key file exists
	privateKeyPath := filepath.Join(tmpDir, "test-key_private.pem")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Error("Private key file was not created")
	}

	// Verify private key has restrictive permissions
	info, err := os.Stat(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Private key file has incorrect permissions: %o, want 0600", mode)
	}

	// Verify keys are valid PEM
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(publicKeyData)
	if block == nil || block.Type != "PUBLIC KEY" {
		t.Error("Public key is not valid PEM format")
	}

	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	block, _ = pem.Decode(privateKeyData)
	if block == nil || block.Type != "PRIVATE KEY" {
		t.Error("Private key is not valid PEM format")
	}
}

func TestGenerateKeysAutoID(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "keys-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set flags without key ID
	keysFlags.output = tmpDir
	keysFlags.keyID = "" // Auto-generate
	keysFlags.format = "pem"
	keysFlags.noConfirm = true

	// Generate keys
	err = generateKeys(nil, []string{})
	if err != nil {
		t.Fatalf("generateKeys() with auto ID error = %v", err)
	}

	// Verify files were created (with auto-generated ID)
	files, err := filepath.Glob(filepath.Join(tmpDir, "*_public.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 public key file, found %d", len(files))
	}
}

func TestSavePublicKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "keys-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test key
	publicKey, _, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Save public key
	path := filepath.Join(tmpDir, "test_public.pem")
	err = savePublicKey(path, publicKey)
	if err != nil {
		t.Fatalf("savePublicKey() error = %v", err)
	}

	// Verify file exists and is valid PEM
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Error("Saved public key is not valid PEM")
	}
	if block.Type != "PUBLIC KEY" {
		t.Errorf("PEM block type = %q, want %q", block.Type, "PUBLIC KEY")
	}

	// Verify key data matches
	if len(block.Bytes) != ed25519.PublicKeySize {
		t.Errorf("Public key size = %d, want %d", len(block.Bytes), ed25519.PublicKeySize)
	}
}

func TestSavePrivateKey(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "keys-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Generate test key
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}

	// Save private key
	path := filepath.Join(tmpDir, "test_private.pem")
	err = savePrivateKey(path, privateKey)
	if err != nil {
		t.Fatalf("savePrivateKey() error = %v", err)
	}

	// Verify file exists and has correct permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Private key permissions = %o, want 0600", info.Mode().Perm())
	}

	// Verify valid PEM
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		t.Error("Saved private key is not valid PEM")
	}
	if block.Type != "PRIVATE KEY" {
		t.Errorf("PEM block type = %q, want %q", block.Type, "PRIVATE KEY")
	}

	// Verify key data matches
	if len(block.Bytes) != ed25519.PrivateKeySize {
		t.Errorf("Private key size = %d, want %d", len(block.Bytes), ed25519.PrivateKeySize)
	}
}

func TestListKeys(t *testing.T) {
	// Just verify it doesn't panic
	err := listKeys(nil, []string{})
	if err != nil {
		t.Errorf("listKeys() error = %v", err)
	}
}
