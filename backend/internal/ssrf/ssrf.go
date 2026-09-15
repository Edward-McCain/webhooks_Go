package ssrf

import (
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync/atomic"
)

var allowPrivate atomic.Bool

// SetAllowPrivateTargets toggles whether private/loopback destinations are permitted.
// Intended for local development and demo mode only.
func SetAllowPrivateTargets(allow bool) {
	allowPrivate.Store(allow)
}

// ValidateTargetURL ensures a webhook destination is not an internal or metadata address.
func ValidateTargetURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("URL is empty")
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("only http and https schemes are allowed")
	}
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}

	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("host is required")
	}

	lower := strings.ToLower(host)
	if !allowPrivate.Load() {
		if lower == "localhost" || strings.HasSuffix(lower, ".localhost") || lower == "metadata.google.internal" {
			return fmt.Errorf("host is not allowed")
		}
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("IP address is not allowed")
		}
		return nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("DNS lookup failed: %w", err)
	}
	if len(ips) == 0 {
		return fmt.Errorf("DNS lookup returned no addresses")
	}
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return fmt.Errorf("resolved IP address is not allowed")
		}
	}
	return nil
}

// ResolveAndValidate resolves the host and re-checks that all IPs are public.
func ResolveAndValidate(host string) ([]net.IP, error) {
	if host == "" {
		return nil, fmt.Errorf("host is required")
	}
	lower := strings.ToLower(host)
	if !allowPrivate.Load() {
		if lower == "localhost" || strings.HasSuffix(lower, ".localhost") {
			return nil, fmt.Errorf("host is not allowed")
		}
	}

	if ip := net.ParseIP(host); ip != nil {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("IP address is not allowed")
		}
		return []net.IP{ip}, nil
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("DNS lookup failed: %w", err)
	}
	out := make([]net.IP, 0, len(ips))
	for _, ip := range ips {
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("resolved IP address is not allowed")
		}
		out = append(out, ip)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no allowed addresses")
	}
	return out, nil
}

func isBlockedIP(ip net.IP) bool {
	if allowPrivate.Load() {
		// Still block cloud metadata even in dev.
		if ip.Equal(net.ParseIP("169.254.169.254")) {
			return true
		}
		return false
	}

	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsUnspecified() || ip.IsInterfaceLocalMulticast() {
		return true
	}

	blocked := []string{
		"169.254.169.254/32",
		"169.254.0.0/16",
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"0.0.0.0/8",
		"100.64.0.0/10",
		"192.0.0.0/24",
		"192.0.2.0/24",
		"198.51.100.0/24",
		"203.0.113.0/24",
		"224.0.0.0/4",
		"240.0.0.0/4",
		"::1/128",
		"fc00::/7",
		"fe80::/10",
		"ff00::/8",
	}

	for _, cidr := range blocked {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		if network.Contains(ip) {
			return true
		}
	}
	return false
}
