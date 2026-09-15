package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	KeyPrefix = "hf_"
	KeyBytes  = 32
)

// GenerateAPIKey creates a new API key and returns the plaintext once, plus prefix and hash.
func GenerateAPIKey() (plaintext, prefix, hash string, err error) {
	buf := make([]byte, KeyBytes)
	if _, err = rand.Read(buf); err != nil {
		return "", "", "", fmt.Errorf("generate api key: %w", err)
	}
	plaintext = KeyPrefix + base64.RawURLEncoding.EncodeToString(buf)
	prefix = plaintext
	if len(prefix) > 11 {
		prefix = plaintext[:11]
	}
	hash = HashAPIKey(plaintext)
	return plaintext, prefix, hash, nil
}

// HashAPIKey returns a SHA-256 hex digest of the API key.
func HashAPIKey(plaintext string) string {
	sum := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(sum[:])
}

// EqualHash compares a plaintext key against a stored hash in constant time.
func EqualHash(plaintext, hash string) bool {
	expected := HashAPIKey(plaintext)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(hash)) == 1
}

// BearerToken extracts the token from an Authorization header.
func BearerToken(header string) (string, bool) {
	header = strings.TrimSpace(header)
	if header == "" {
		return "", false
	}
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(header[len(prefix):])
	if token == "" {
		return "", false
	}
	return token, true
}

// GenerateEndpointSecret creates a random webhook signing secret.
func GenerateEndpointSecret() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "whsec_" + base64.RawURLEncoding.EncodeToString(buf), nil
}

// GeneratePublicID creates a public endpoint identifier.
func GeneratePublicID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "ep_" + hex.EncodeToString(buf), nil
}

// GenerateEventID creates an external event ID when the client omits one.
func GenerateEventID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "evt_" + hex.EncodeToString(buf), nil
}
