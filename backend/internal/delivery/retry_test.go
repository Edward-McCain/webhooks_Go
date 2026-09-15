package delivery_test

import (
	"testing"
	"time"

	"github.com/Edward-McCain/webhooks_Go/backend/internal/delivery"
)

func TestBackoffSchedule(t *testing.T) {
	cases := []struct {
		attempt int
		min     time.Duration
		max     time.Duration
	}{
		{1, 8 * time.Second, 12 * time.Second},
		{2, 24 * time.Second, 36 * time.Second},
		{3, 96 * time.Second, 144 * time.Second},
	}

	for _, tc := range cases {
		d := delivery.Backoff(tc.attempt, delivery.DefaultBackoffSchedule, 0.2)
		if d < tc.min || d > tc.max {
			t.Fatalf("attempt %d: got %v, want between %v and %v", tc.attempt, d, tc.min, tc.max)
		}
	}
}

func TestIsRetryableStatus(t *testing.T) {
	retryable := []int{408, 429, 500, 502, 503, 504}
	for _, code := range retryable {
		if !delivery.IsRetryableStatus(code) {
			t.Fatalf("expected %d to be retryable", code)
		}
	}
	nonRetryable := []int{400, 401, 403, 404, 422}
	for _, code := range nonRetryable {
		if delivery.IsRetryableStatus(code) {
			t.Fatalf("expected %d not to be retryable", code)
		}
	}
}
