package config

import (
	"net/netip"
	"testing"
)

func TestLoadRequiresCredentials(t *testing.T) {
	t.Setenv("AUTH_USER", "")
	t.Setenv("AUTH_PASSWORD_HASH", "")
	t.Setenv("AUTH_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without credentials")
	}
}

func TestLoadUnescapesComposeDollarAndParsesProxies(t *testing.T) {
	t.Setenv("AUTH_USER", "jakob")
	t.Setenv("AUTH_PASSWORD_HASH", "$$2a$$10$$abc")
	t.Setenv("TRUSTED_PROXIES", "192.168.178.20, 10.0.0.0/8")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthPasswordHash != "$2a$10$abc" {
		t.Fatalf("hash %q", c.AuthPasswordHash)
	}
	if len(c.TrustedProxies) != 2 ||
		!c.TrustedProxies[0].Contains(netip.MustParseAddr("192.168.178.20")) ||
		c.TrustedProxies[0].Contains(netip.MustParseAddr("192.168.178.21")) ||
		!c.TrustedProxies[1].Contains(netip.MustParseAddr("10.1.2.3")) {
		t.Fatalf("proxies %v", c.TrustedProxies)
	}
}

func TestLoadRejectsBadProxy(t *testing.T) {
	t.Setenv("AUTH_USER", "jakob")
	t.Setenv("AUTH_PASSWORD", "secretsecret")
	t.Setenv("TRUSTED_PROXIES", "caddy.lan")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}
