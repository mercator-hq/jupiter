package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var keysFlags struct {
	output    string
	keyID     string
	format    string
	noConfirm bool
}

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Manage cryptographic keys",
	Long: `Generate, rotate, and manage Ed25519 keypairs for evidence signing.

The keys command provides utilities for managing cryptographic keys used
to sign evidence records. Keys are generated using the Ed25519 signature
algorithm for strong security with small key sizes.

Subcommands:
  generate - Generate new Ed25519 keypair
  list     - List all keys (not yet implemented)
  rotate   - Rotate signing key (not yet implemented)

Examples:
  # Generate new keypair
  mercator keys generate

  # Generate with custom key ID
  mercator keys generate --key-id "prod-2025"

  # List active keys
  mercator keys list`,
}

var keysGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate new keypair",
	Long: `Generate a new Ed25519 keypair for evidence signing.

The generated keys are saved to PEM files with restrictive permissions:
  - Public key:  0644 (readable by all)
  - Private key: 0600 (readable only by owner)

Examples:
  # Generate keypair with auto-generated ID
  mercator keys generate

  # Generate with custom ID
  mercator keys generate --key-id "prod-2025-11"

  # Save to custom directory
  mercator keys generate --output /etc/mercator/keys`,
	RunE: generateKeys,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all keys",
	Long:  `List all cryptographic keys with metadata.`,
	RunE:  listKeys,
}

func init() {
	rootCmd.AddCommand(keysCmd)
	keysCmd.AddCommand(keysGenerateCmd, keysListCmd)

	keysGenerateCmd.Flags().StringVarP(&keysFlags.output, "output", "o", "./keys", "output directory")
	keysGenerateCmd.Flags().StringVar(&keysFlags.keyID, "key-id", "", "key ID (auto-generated if empty)")
	keysGenerateCmd.Flags().StringVar(&keysFlags.format, "format", "pem", "output format: pem, base64, hex")
	keysGenerateCmd.Flags().BoolVar(&keysFlags.noConfirm, "no-confirm", false, "skip confirmation prompts")
}

func generateKeys(cmd *cobra.Command, args []string) error {
	// Generate key ID if not provided
	if keysFlags.keyID == "" {
		keysFlags.keyID = fmt.Sprintf("key-%d", time.Now().Unix())
	}

	fmt.Println("Generating Ed25519 keypair...")
	fmt.Println()

	// Generate Ed25519 keypair
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate keypair: %w", err)
	}

	// Create output directory with restricted permissions (0750)
	if err := os.MkdirAll(keysFlags.output, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Save public key
	publicKeyPath := filepath.Join(keysFlags.output, keysFlags.keyID+"_public.pem")
	if err := savePublicKey(publicKeyPath, publicKey); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	// Save private key
	privateKeyPath := filepath.Join(keysFlags.output, keysFlags.keyID+"_private.pem")
	if err := savePrivateKey(privateKeyPath, privateKey); err != nil {
		return fmt.Errorf("failed to save private key: %w", err)
	}

	fmt.Printf("Key ID: %s\n", keysFlags.keyID)
	fmt.Printf("Public Key:  %s\n", publicKeyPath)
	fmt.Printf("Private Key: %s\n", privateKeyPath)
	fmt.Println()
	fmt.Println("⚠️  Warning: Store private key securely and never commit to version control")
	fmt.Println("✓  Keys generated successfully")
	fmt.Println()
	fmt.Println("Configuration snippet:")
	fmt.Println("evidence:")
	fmt.Printf("  signing_key_path: \"%s\"\n", privateKeyPath)

	return nil
}

func savePublicKey(path string, key ed25519.PublicKey) error {
	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: key,
	}

	// #nosec G304 G302 - User-specified output path for public key is expected behavior for a CLI tool.
	// Public key file permissions (0644) are intentionally world-readable as this is a public key.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

func savePrivateKey(path string, key ed25519.PrivateKey) error {
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: key,
	}

	// #nosec G304 - User-specified output path for private key is expected behavior for a CLI tool.
	// File permissions (0600) are correctly restricted to owner-only access.
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer file.Close()

	return pem.Encode(file, block)
}

func listKeys(cmd *cobra.Command, args []string) error {
	fmt.Println("Key listing not yet implemented")
	fmt.Println()
	fmt.Println("This feature will be implemented in a future release.")
	fmt.Println("For now, you can list keys manually:")
	fmt.Println("  ls -la keys/")
	return nil
}
