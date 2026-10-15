// Package config liest die Konfiguration aus Umgebungsvariablen.
package config

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr             string
	DataDir          string
	AuthUser         string
	AuthPasswordHash string
	// AuthPasswordPlain ist gesetzt, wenn AUTH_PASSWORD (Klartext) verwendet wurde.
	AuthPasswordPlain string
	SessionTTL        time.Duration
	// TrustedProxies: Nur von diesen Adressen werden X-Forwarded-For/-Proto ausgewertet
	// (z. B. der Caddy-Reverse-Proxy).
	TrustedProxies []netip.Prefix
	MaxUploadBytes int64
}

func env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return def
}

// Load liest die Konfiguration; fehlende Zugangsdaten sind ein Fehler.
func Load() (Config, error) {
	c := Config{
		Addr:              env("ADDR", ":8080"),
		DataDir:           env("DATA_DIR", "./data"),
		AuthUser:          env("AUTH_USER", ""),
		AuthPasswordHash:  env("AUTH_PASSWORD_HASH", ""),
		AuthPasswordPlain: env("AUTH_PASSWORD", ""),
	}
	// In docker-compose.yml muss `$` im Hash als `$$` geschrieben werden; wer das
	// vergisst, bekommt einen kaputten Hash. Einfache Hilfe: `$$` wieder zu `$`.
	c.AuthPasswordHash = strings.ReplaceAll(c.AuthPasswordHash, "$$", "$")

	days, err := strconv.Atoi(env("SESSION_DAYS", "30"))
	if err != nil || days < 1 {
		return c, fmt.Errorf("SESSION_DAYS: invalid value")
	}
	c.SessionTTL = time.Duration(days) * 24 * time.Hour

	mb, err := strconv.ParseInt(env("MAX_UPLOAD_MB", "500"), 10, 64)
	if err != nil || mb < 1 {
		return c, fmt.Errorf("MAX_UPLOAD_MB: invalid value")
	}
	c.MaxUploadBytes = mb << 20

	c.TrustedProxies, err = ParsePrefixes(env("TRUSTED_PROXIES", ""))
	if err != nil {
		return c, fmt.Errorf("TRUSTED_PROXIES: %w", err)
	}

	if c.AuthUser == "" || (c.AuthPasswordHash == "" && c.AuthPasswordPlain == "") {
		return c, errors.New("AUTH_USER and AUTH_PASSWORD_HASH (or AUTH_PASSWORD) must be set – the app does not run without login")
	}
	return c, nil
}

// ParsePrefixes liest eine kommagetrennte Liste von IPs oder CIDR-Bereichen.
func ParsePrefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "/") {
			p, err := netip.ParsePrefix(part)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q", part)
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(part)
		if err != nil {
			return nil, fmt.Errorf("invalid IP %q", part)
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}
