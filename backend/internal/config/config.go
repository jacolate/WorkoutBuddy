// Package config liest die Konfiguration aus Umgebungsvariablen.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type TLSMode string

const (
	TLSOff        TLSMode = "off"
	TLSSelfSigned TLSMode = "self-signed"
	TLSFiles      TLSMode = "files"
)

type Config struct {
	Addr             string
	DataDir          string
	AuthUser         string
	AuthPasswordHash string
	// AuthPasswordPlain ist gesetzt, wenn AUTH_PASSWORD (Klartext) verwendet wurde.
	AuthPasswordPlain string
	SessionTTL        time.Duration
	TrustProxy        bool
	TLSMode           TLSMode
	TLSCert           string
	TLSKey            string
	TLSHosts          []string
	MaxUploadBytes    int64
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
		TLSCert:           env("TLS_CERT", ""),
		TLSKey:            env("TLS_KEY", ""),
	}
	// In docker-compose.yml muss `$` im Hash als `$$` geschrieben werden; wer das
	// vergisst, bekommt einen kaputten Hash. Einfache Hilfe: `$$` wieder zu `$`.
	c.AuthPasswordHash = strings.ReplaceAll(c.AuthPasswordHash, "$$", "$")

	days, err := strconv.Atoi(env("SESSION_DAYS", "30"))
	if err != nil || days < 1 {
		return c, fmt.Errorf("SESSION_DAYS: invalid value")
	}
	c.SessionTTL = time.Duration(days) * 24 * time.Hour

	c.TrustProxy, _ = strconv.ParseBool(env("TRUST_PROXY", "false"))

	mb, err := strconv.ParseInt(env("MAX_UPLOAD_MB", "500"), 10, 64)
	if err != nil || mb < 1 {
		return c, fmt.Errorf("MAX_UPLOAD_MB: invalid value")
	}
	c.MaxUploadBytes = mb << 20

	switch mode := TLSMode(env("TLS_MODE", "")); {
	case mode == "" && c.TLSCert != "":
		c.TLSMode = TLSFiles
	case mode == "" || mode == TLSOff:
		c.TLSMode = TLSOff
	case mode == TLSSelfSigned || mode == TLSFiles:
		c.TLSMode = mode
	default:
		return c, fmt.Errorf("TLS_MODE: expected off, self-signed or files")
	}
	if c.TLSMode == TLSFiles && (c.TLSCert == "" || c.TLSKey == "") {
		return c, errors.New("TLS_MODE=files requires TLS_CERT and TLS_KEY")
	}
	if hosts := env("TLS_HOSTS", ""); hosts != "" {
		for _, h := range strings.Split(hosts, ",") {
			if h = strings.TrimSpace(h); h != "" {
				c.TLSHosts = append(c.TLSHosts, h)
			}
		}
	}

	if c.AuthUser == "" || (c.AuthPasswordHash == "" && c.AuthPasswordPlain == "") {
		return c, errors.New("AUTH_USER and AUTH_PASSWORD_HASH (or AUTH_PASSWORD) must be set – the app does not run without login")
	}
	return c, nil
}
