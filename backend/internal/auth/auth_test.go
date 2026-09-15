package auth_test

import (
	"strings"
	"testing"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/auth"
)

func TestGenerateAndHashAPIKey(t *testing.T) {
	plain, prefix, hash, err := auth.GenerateAPIKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(plain, "hf_") {
		t.Fatalf("unexpected key prefix: %s", plain)
	}
	if prefix == "" || hash == "" {
		t.Fatal("prefix and hash required")
	}
	if !auth.EqualHash(plain, hash) {
		t.Fatal("hash mismatch")
	}
	if auth.EqualHash(plain+"x", hash) {
		t.Fatal("different key should not match")
	}
}

func TestBearerToken(t *testing.T) {
	tok, ok := auth.BearerToken("Bearer abc.def")
	if !ok || tok != "abc.def" {
		t.Fatalf("got %q %v", tok, ok)
	}
	if _, ok := auth.BearerToken("Basic abc"); ok {
		t.Fatal("expected failure for non-bearer")
	}
}
