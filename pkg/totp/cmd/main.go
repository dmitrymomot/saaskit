package main

import (
	"fmt"
	"log"

	"github.com/dmitrymomot/saaskit/pkg/totp"
)

func main() {
	// Generate a base64-encoded encryption key for environment variables
	encodedKey, err := totp.GenerateEncodedEncryptionKey()
	if err != nil {
		log.Fatalf("Failed to generate encoded encryption key: %v", err)
	}

	fmt.Printf("Generated Encoded Encryption Key (for TOTP_ENCRYPTION_KEY env var): \n———\n%s\n———\n", encodedKey)
}
