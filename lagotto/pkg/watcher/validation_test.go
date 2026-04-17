package watcher

import (
	"testing"
)

func TestValidateWebhookURL_AcceptsHTTPS(t *testing.T) {
	urls := []string{
		"https://webhook.example.com/notify/abc123",
		"https://example.com/webhook",
		"https://my-server.example.org:8443/notify",
	}
	for _, u := range urls {
		if err := ValidateWebhookURL(u); err != nil {
			t.Errorf("expected %q to be accepted, got error: %v", u, err)
		}
	}
}

func TestValidateWebhookURL_RejectsHTTP(t *testing.T) {
	if err := ValidateWebhookURL("http://example.com/webhook"); err == nil {
		t.Error("expected http:// to be rejected")
	}
}

func TestValidateWebhookURL_RejectsMetadataService(t *testing.T) {
	// EC2 Instance Metadata Service — primary SSRF target
	blocked := []string{
		"https://169.254.169.254/latest/meta-data/iam/security-credentials/",
		"https://169.254.169.254:80/foo",
		"https://169.254.0.1/anything",
	}
	for _, u := range blocked {
		if err := ValidateWebhookURL(u); err == nil {
			t.Errorf("expected %q (link-local) to be rejected", u)
		}
	}
}

func TestValidateWebhookURL_RejectsLoopback(t *testing.T) {
	blocked := []string{
		"https://127.0.0.1/admin",
		"https://localhost/admin",
		"https://[::1]/admin",
	}
	for _, u := range blocked {
		if err := ValidateWebhookURL(u); err == nil {
			t.Errorf("expected %q (loopback) to be rejected", u)
		}
	}
}

func TestValidateWebhookURL_RejectsPrivateRanges(t *testing.T) {
	blocked := []string{
		"https://10.0.0.1/internal",
		"https://172.16.0.1/internal",
		"https://192.168.1.1/internal",
	}
	for _, u := range blocked {
		if err := ValidateWebhookURL(u); err == nil {
			t.Errorf("expected %q (private IP) to be rejected", u)
		}
	}
}

func TestValidateWebhookURL_RejectsNoScheme(t *testing.T) {
	if err := ValidateWebhookURL("example.com/webhook"); err == nil {
		t.Error("expected URL without scheme to be rejected")
	}
}

func TestValidateWebhookURL_RejectsEmptyHost(t *testing.T) {
	if err := ValidateWebhookURL("https:///webhook"); err == nil {
		t.Error("expected URL with empty host to be rejected")
	}
}
