package main

import (
	"github.com/spf13/cobra"
)

var certsCmd = &cobra.Command{
	Use:   "certs",
	Short: "Manage TLS certificates",
	Long: `Manage TLS certificates for Mercator Jupiter.

The certs command provides utilities for managing TLS certificates used
for secure communication. This includes validation, inspection, and
generation of certificates for testing.

Subcommands:
  validate - Validate certificate and key pair
  info     - Display certificate details
  generate - Generate self-signed certificate for testing

Examples:
  # Validate certificate and key
  mercator certs validate --cert server.crt --key server.key

  # Display certificate information
  mercator certs info server.crt

  # Generate self-signed certificate for testing
  mercator certs generate --host localhost`,
}

func init() {
	rootCmd.AddCommand(certsCmd)
}
