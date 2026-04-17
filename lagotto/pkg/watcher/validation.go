package watcher

import (
	"fmt"
	"net"
	"net/url"
)

// ValidateWebhookURL checks that a webhook URL is safe to call from a Lambda.
// Rejects non-HTTPS schemes and blocks private/internal IP ranges to prevent SSRF.
func ValidateWebhookURL(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("webhook URL must use HTTPS (got %q)", u.Scheme)
	}
	if u.Host == "" {
		return fmt.Errorf("webhook URL must have a host")
	}

	host := u.Hostname()

	// Reject bare IPs in the URL that are private/internal
	if ip := net.ParseIP(host); ip != nil {
		return checkIP(ip)
	}

	// Resolve hostname and check all resolved IPs
	// Skip resolution errors at storage time — the notifier re-validates at send time
	ips, err := net.LookupHost(host)
	if err != nil {
		return nil
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if err := checkIP(ip); err != nil {
			return fmt.Errorf("webhook host %q resolves to blocked address: %w", host, err)
		}
	}
	return nil
}

func checkIP(ip net.IP) error {
	if ip.IsLoopback() {
		return fmt.Errorf("webhook must not target loopback addresses")
	}
	if ip.IsPrivate() {
		return fmt.Errorf("webhook must not target private network addresses")
	}
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return fmt.Errorf("webhook must not target link-local addresses (includes EC2 metadata service)")
	}
	if ip.IsUnspecified() {
		return fmt.Errorf("webhook must not target unspecified addresses")
	}
	return nil
}
