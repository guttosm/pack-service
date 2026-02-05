//go:build ignore

// This script generates secure random keys for JWT authentication.
// Run with: go run scripts/generate_keys.go
package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
)

func generateSecureKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

func main() {
	fmt.Println("=== Pack Service Key Generator ===")
	fmt.Println()

	// Generate JWT Secret Key (32 bytes = 256 bits)
	jwtSecret, err := generateSecureKey(32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JWT secret: %v\n", err)
		os.Exit(1)
	}

	// Generate JWT Refresh Secret Key (32 bytes = 256 bits)
	jwtRefreshSecret, err := generateSecureKey(32)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating JWT refresh secret: %v\n", err)
		os.Exit(1)
	}

	// Generate API Key (24 bytes)
	apiKey, err := generateSecureKey(24)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating API key: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Add these to your .env file:")
	fmt.Println()
	fmt.Println("# JWT Configuration")
	fmt.Printf("JWT_SECRET_KEY=%s\n", jwtSecret)
	fmt.Printf("JWT_REFRESH_SECRET_KEY=%s\n", jwtRefreshSecret)
	fmt.Println()
	fmt.Println("# API Key (optional, for API key authentication)")
	fmt.Printf("API_KEYS=%s\n", apiKey)
	fmt.Println()
	fmt.Println("=== IMPORTANT ===")
	fmt.Println("- Never commit these keys to version control")
	fmt.Println("- Use different keys for each environment (dev, staging, prod)")
	fmt.Println("- Store production keys in a secure secret manager")
}
