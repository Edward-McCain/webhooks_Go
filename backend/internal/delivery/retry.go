package delivery

import (
	"math"
	"math/rand"
	"time"
)

// DefaultBackoffSchedule mirrors a production-like retry cadence.
var DefaultBackoffSchedule = []time.Duration{
	10 * time.Second,
	30 * time.Second,
	2 * time.Minute,
	5 * time.Minute,
	15 * time.Minute,
}

// Backoff calculates the delay before the next retry for the given 1-based attempt number.
// attempt=1 means the first retry after the initial failure.
func Backoff(attempt int, schedule []time.Duration, jitterRatio float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if len(schedule) == 0 {
		schedule = DefaultBackoffSchedule
	}

	idx := attempt - 1
	var base time.Duration
	if idx < len(schedule) {
		base = schedule[idx]
	} else {
		base = schedule[len(schedule)-1]
		// Mild exponential growth beyond the schedule.
		extra := idx - (len(schedule) - 1)
		base = time.Duration(float64(base) * math.Pow(2, float64(extra)))
		if base > 30*time.Minute {
			base = 30 * time.Minute
		}
	}

	if jitterRatio <= 0 {
		return base
	}
	if jitterRatio > 1 {
		jitterRatio = 1
	}

	jitter := time.Duration(float64(base) * jitterRatio * (rand.Float64()*2 - 1))
	delay := base + jitter
	if delay < 0 {
		delay = 0
	}
	return delay
}

// IsRetryableStatus reports whether an HTTP status should be retried.
func IsRetryableStatus(status int) bool {
	switch status {
	case 408, 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// IsRetryableError reports whether a network-level failure should be retried.
func IsRetryableError(err error) bool {
	return err != nil
}
