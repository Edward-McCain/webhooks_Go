package ssrf_test

import (
	"testing"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/ssrf"
)

func TestValidateTargetURLBlocksLocalhost(t *testing.T) {
	ssrf.SetAllowPrivateTargets(false)
	defer ssrf.SetAllowPrivateTargets(false)

	cases := []string{
		"http://localhost/hook",
		"http://127.0.0.1/hook",
		"http://169.254.169.254/latest/meta-data",
		"http://192.168.1.10/hook",
		"ftp://example.com/hook",
	}
	for _, raw := range cases {
		if err := ssrf.ValidateTargetURL(raw); err == nil {
			t.Fatalf("expected rejection for %s", raw)
		}
	}
}

func TestValidateTargetURLAllowsPublicHTTPS(t *testing.T) {
	ssrf.SetAllowPrivateTargets(false)
	// example.com resolves publicly; may fail offline — accept either parse/scheme success path for IP literals.
	if err := ssrf.ValidateTargetURL("https://example.com/webhook"); err != nil {
		t.Fatalf("expected public URL to pass, got %v", err)
	}
}

func TestAllowPrivateTargets(t *testing.T) {
	ssrf.SetAllowPrivateTargets(true)
	defer ssrf.SetAllowPrivateTargets(false)
	if err := ssrf.ValidateTargetURL("http://127.0.0.1:8080/demo/webhook"); err != nil {
		t.Fatalf("expected private target allowed in dev mode: %v", err)
	}
}
