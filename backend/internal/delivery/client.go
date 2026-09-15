package delivery

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	hmacsig "github.com/Edward-McCain/webhooks_Go/backend/internal/hmac"
	"github.com/Edward-McCain/webhooks_Go/backend/internal/ssrf"
)

// Result captures the outcome of a single HTTP delivery attempt.
type Result struct {
	StatusCode int
	Body       string
	Duration   time.Duration
	Err        error
	Retryable  bool
}

// Client delivers webhooks to external endpoints.
type Client struct {
	httpClient     *http.Client
	maxResponseLen int64
	timeout        time.Duration
}

func NewClient(timeout time.Duration, maxResponseLen int64) *Client {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if maxResponseLen <= 0 {
		maxResponseLen = 64 << 10
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: timeout,
	}

	return &Client{
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 3 {
					return fmt.Errorf("too many redirects")
				}
				return ssrf.ValidateTargetURL(req.URL.String())
			},
		},
		maxResponseLen: maxResponseLen,
		timeout:        timeout,
	}
}

// Deliver sends the payload to targetURL with HMAC signature headers.
func (c *Client) Deliver(ctx context.Context, targetURL, secret, eventID, eventType string, payload []byte) Result {
	start := time.Now()

	if err := ssrf.ValidateTargetURL(targetURL); err != nil {
		return Result{Duration: time.Since(start), Err: err, Retryable: false}
	}

	u, err := url.Parse(targetURL)
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Retryable: false}
	}
	if _, err := ssrf.ResolveAndValidate(u.Hostname()); err != nil {
		return Result{Duration: time.Since(start), Err: err, Retryable: false}
	}

	ts := time.Now().UTC().Unix()
	sig := hmacsig.Sign(secret, ts, payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return Result{Duration: time.Since(start), Err: err, Retryable: false}
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HookForge/1.0")
	req.Header.Set("X-Event-ID", eventID)
	req.Header.Set("X-Event-Type", eventType)
	req.Header.Set("X-Webhook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Webhook-Signature", sig)

	resp, err := c.httpClient.Do(req)
	duration := time.Since(start)
	if err != nil {
		return Result{Duration: duration, Err: err, Retryable: true}
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, c.maxResponseLen+1)
	body, readErr := io.ReadAll(limited)
	if readErr != nil {
		return Result{
			StatusCode: resp.StatusCode,
			Duration:   duration,
			Err:        readErr,
			Retryable:  true,
		}
	}
	if int64(len(body)) > c.maxResponseLen {
		body = body[:c.maxResponseLen]
	}

	retryable := IsRetryableStatus(resp.StatusCode)
	var resultErr error
	if resp.StatusCode >= 400 {
		resultErr = fmt.Errorf("upstream status %d", resp.StatusCode)
	}

	return Result{
		StatusCode: resp.StatusCode,
		Body:       string(body),
		Duration:   duration,
		Err:        resultErr,
		Retryable:  retryable,
	}
}
