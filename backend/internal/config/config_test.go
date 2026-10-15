package config

import "testing"

func TestLoadRequiresCredentials(t *testing.T) {
	t.Setenv("AUTH_USER", "")
	t.Setenv("AUTH_PASSWORD_HASH", "")
	t.Setenv("AUTH_PASSWORD", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error without credentials")
	}
}

func TestLoadUnescapesComposeDollar(t *testing.T) {
	t.Setenv("AUTH_USER", "jakob")
	t.Setenv("AUTH_PASSWORD_HASH", "$$2a$$10$$abc")
	t.Setenv("TLS_MODE", "self-signed")
	t.Setenv("TLS_HOSTS", "192.168.1.10, workout.lan")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.AuthPasswordHash != "$2a$10$abc" {
		t.Fatalf("hash %q", c.AuthPasswordHash)
	}
	if c.TLSMode != TLSSelfSigned || len(c.TLSHosts) != 2 || c.TLSHosts[1] != "workout.lan" {
		t.Fatalf("tls config %+v", c)
	}
}

func TestLoadRejectsBadTLSMode(t *testing.T) {
	t.Setenv("AUTH_USER", "jakob")
	t.Setenv("AUTH_PASSWORD", "secretsecret")
	t.Setenv("TLS_MODE", "maybe")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}
