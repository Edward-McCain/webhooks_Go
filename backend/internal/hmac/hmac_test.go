package hmacsig_test

import (
	"fmt"
	"testing"
	"time"

	hmacsig "github.com/Edward-McCain/webhooks_Go/backend/internal/hmac"
)

func TestSignAndVerify(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"hello":"world"}`)
	ts := time.Now().UTC().Unix()
	sig := hmacsig.Sign(secret, ts, payload)

	err := hmacsig.Verify(secret, sig, fmt.Sprintf("%d", ts), payload, time.Now().UTC(), hmacsig.DefaultTolerance)
	if err != nil {
		t.Fatalf("expected valid signature, got %v", err)
	}
}

func TestVerifyRejectsBadSignature(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"hello":"world"}`)
	ts := time.Now().UTC().Unix()

	err := hmacsig.Verify(secret, "deadbeef", fmt.Sprintf("%d", ts), payload, time.Now().UTC(), hmacsig.DefaultTolerance)
	if err == nil {
		t.Fatal("expected signature mismatch")
	}
}

func TestVerifyRejectsStaleTimestamp(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"hello":"world"}`)
	ts := time.Now().UTC().Add(-20 * time.Minute).Unix()
	sig := hmacsig.Sign(secret, ts, payload)

	err := hmacsig.Verify(secret, sig, fmt.Sprintf("%d", ts), payload, time.Now().UTC(), hmacsig.DefaultTolerance)
	if err == nil {
		t.Fatal("expected stale timestamp rejection")
	}
}
