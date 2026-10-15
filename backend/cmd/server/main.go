// Command server startet die Workout-App.
//
// Subcommands:
//
//	server                 startet den Server (Standard)
//	server hash-password   erzeugt einen bcrypt-Hash für AUTH_PASSWORD_HASH
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/term"

	"workoutapp/internal/api"
	"workoutapp/internal/auth"
	"workoutapp/internal/config"
	"workoutapp/internal/media"
	"workoutapp/internal/store"
	"workoutapp/internal/tlsutil"
	"workoutapp/web"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "hash-password":
			if err := hashPassword(); err != nil {
				fmt.Fprintln(os.Stderr, "Fehler:", err)
				os.Exit(1)
			}
			return
		case "healthcheck":
			os.Exit(healthcheck())
		default:
			fmt.Fprintf(os.Stderr, "unbekannter Befehl %q (erlaubt: hash-password, healthcheck)\n", os.Args[1])
			os.Exit(2)
		}
	}
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.AuthPasswordHash == "" {
		slog.Warn("AUTH_PASSWORD im Klartext gesetzt – besser AUTH_PASSWORD_HASH verwenden (siehe `hash-password`)")
		h, err := auth.HashPassword(cfg.AuthPasswordPlain)
		if err != nil {
			return err
		}
		cfg.AuthPasswordHash = h
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}
	st, err := store.Open(filepath.Join(cfg.DataDir, "workout.db"))
	if err != nil {
		return err
	}
	defer st.Close()

	ms, err := media.New(filepath.Join(cfg.DataDir, "recordings"))
	if err != nil {
		return err
	}
	authSvc, err := auth.New(st, auth.Options{
		User:         cfg.AuthUser,
		PasswordHash: cfg.AuthPasswordHash,
		SessionTTL:   cfg.SessionTTL,
		TrustProxy:   cfg.TrustProxy,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go authSvc.PurgeLoop(ctx, time.Hour)

	handler := api.New(api.Options{
		Store:          st,
		Media:          ms,
		Auth:           authSvc,
		MaxUploadBytes: cfg.MaxUploadBytes,
		Static:         web.Dist(),
	})

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	errc := make(chan error, 1)
	go func() {
		switch cfg.TLSMode {
		case config.TLSSelfSigned:
			cert, err := tlsutil.LoadOrCreate(filepath.Join(cfg.DataDir, "tls"), cfg.TLSHosts)
			if err != nil {
				errc <- err
				return
			}
			srv.TLSConfig = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
			slog.Info("listening (HTTPS, self-signed)", "addr", cfg.Addr)
			errc <- srv.ListenAndServeTLS("", "")
		case config.TLSFiles:
			slog.Info("listening (HTTPS)", "addr", cfg.Addr)
			errc <- srv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey)
		default:
			slog.Info("listening (HTTP)", "addr", cfg.Addr)
			errc <- srv.ListenAndServe()
		}
	}()

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// hashPassword liest ein Passwort (verdeckt, falls Terminal) und gibt den bcrypt-Hash aus.
func hashPassword() error {
	var pw string
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprint(os.Stderr, "Passwort: ")
		b1, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		fmt.Fprint(os.Stderr, "Wiederholen: ")
		b2, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return err
		}
		if string(b1) != string(b2) {
			return errors.New("Passwörter stimmen nicht überein")
		}
		pw = string(b1)
	} else {
		line, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && line == "" {
			return err
		}
		pw = strings.TrimRight(line, "\r\n")
	}
	if len(pw) < 8 {
		return errors.New("Passwort muss mindestens 8 Zeichen haben")
	}
	h, err := auth.HashPassword(pw)
	if err != nil {
		return err
	}
	fmt.Println(h)
	fmt.Fprintln(os.Stderr, "\nIn die .env-Datei übernehmen (einfache Anführungszeichen wegen der $-Zeichen):")
	fmt.Fprintln(os.Stderr, "  AUTH_PASSWORD_HASH='"+h+"'")
	fmt.Fprintln(os.Stderr, "Direkt in docker-compose.yml stattdessen jedes $ als $$ schreiben:")
	fmt.Fprintln(os.Stderr, "  AUTH_PASSWORD_HASH: \""+strings.ReplaceAll(h, "$", "$$")+"\"")
	return nil
}

// healthcheck wird vom Docker-HEALTHCHECK aufgerufen (distroless hat kein curl).
func healthcheck() int {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	scheme := "http"
	mode := os.Getenv("TLS_MODE")
	if mode == "self-signed" || mode == "files" || os.Getenv("TLS_CERT") != "" {
		scheme = "https"
	}
	client := &http.Client{
		Timeout:   3 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	resp, err := client.Get(scheme + "://" + addr + "/api/health")
	if err != nil {
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
