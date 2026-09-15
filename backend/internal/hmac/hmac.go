package hmacsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const DefaultTolerance = 5 * time.Minute

// Sign creates an HMAC-SHA256 signature for timestamp + "." + payload.
func Sign(secret string, timestamp int64, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(fmt.Sprintf("%d.", timestamp)))
	_, _ = mac.Write(payload)
	return hex.EncodeToString(mac.Sum(nil))
}

// Verify checks the signature using constant-time comparison and timestamp freshness.
func Verify(secret, signatureHeader string, timestampHeader string, payload []byte, now time.Time, tolerance time.Duration) error {
	if secret == "" {
		return fmt.Errorf("empty secret")
	}
	if signatureHeader == "" {
		return fmt.Errorf("missing signature")
	}
	if timestampHeader == "" {
		return fmt.Errorf("missing timestamp")
	}

	ts, err := strconv.ParseInt(timestampHeader, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	eventTime := time.Unix(ts, 0).UTC()
	delta := now.UTC().Sub(eventTime)
	if delta < 0 {
		delta = -delta
	}
	if delta > tolerance {
		return fmt.Errorf("timestamp outside tolerance window")
	}

	expected := Sign(secret, ts, payload)
	provided := normalizeSignature(signatureHeader)

	if !hmac.Equal([]byte(expected), []byte(provided)) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

func normalizeSignature(sig string) string {
	sig = strings.TrimSpace(sig)
	if strings.HasPrefix(strings.ToLower(sig), "sha256=") {
		return strings.TrimPrefix(sig, "sha256=")
	}
	if strings.HasPrefix(strings.ToLower(sig), "v1=") {
		return sig[3:]
	}
	return sig
}
