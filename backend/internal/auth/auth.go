// Package auth implementiert einen einfachen Single-User-Login mit Session-Cookie.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"workoutapp/internal/store"
)

const CookieName = "workout_session"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrRateLimited        = errors.New("too many attempts")
)

// Service prüft Zugangsdaten und verwaltet Sessions.
type Service struct {
	store        *store.Store
	user         string
	passwordHash []byte
	sessionTTL   time.Duration
	trustProxy   bool
	limiter      *limiter
	now          func() time.Time
}

type Options struct {
	User         string
	PasswordHash string
	SessionTTL   time.Duration
	TrustProxy   bool
	// MaxFailures pro Window und IP, danach 429.
	MaxFailures int
	Window      time.Duration
}

func New(st *store.Store, o Options) (*Service, error) {
	if o.User == "" || o.PasswordHash == "" {
		return nil, errors.New("auth: user and password hash required")
	}
	if _, err := bcrypt.Cost([]byte(o.PasswordHash)); err != nil {
		return nil, errors.New("auth: AUTH_PASSWORD_HASH is not a valid bcrypt hash")
	}
	if o.SessionTTL == 0 {
		o.SessionTTL = 30 * 24 * time.Hour
	}
	if o.MaxFailures == 0 {
		o.MaxFailures = 5
	}
	if o.Window == 0 {
		o.Window = time.Minute
	}
	return &Service{
		store:        st,
		user:         o.User,
		passwordHash: []byte(o.PasswordHash),
		sessionTTL:   o.SessionTTL,
		trustProxy:   o.TrustProxy,
		limiter:      newLimiter(o.MaxFailures, o.Window),
		now:          time.Now,
	}, nil
}

// HashPassword erzeugt einen bcrypt-Hash.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(h), err
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// Login prüft die Zugangsdaten und legt bei Erfolg eine Session an.
func (s *Service) Login(ctx context.Context, ip, user, password string) (token string, expires time.Time, err error) {
	if !s.limiter.allow(ip, s.now()) {
		return "", time.Time{}, ErrRateLimited
	}
	// bcrypt läuft immer, damit die Antwortzeit nicht verrät, ob der Benutzername stimmt.
	pwErr := bcrypt.CompareHashAndPassword(s.passwordHash, []byte(password))
	userOK := subtle.ConstantTimeCompare([]byte(strings.ToLower(user)), []byte(strings.ToLower(s.user))) == 1
	if pwErr != nil || !userOK {
		s.limiter.fail(ip, s.now())
		return "", time.Time{}, ErrInvalidCredentials
	}
	s.limiter.reset(ip)

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", time.Time{}, err
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	expires = s.now().Add(s.sessionTTL)
	if err := s.store.CreateAuthSession(ctx, hashToken(token), expires); err != nil {
		return "", time.Time{}, err
	}
	return token, expires, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteAuthSession(ctx, hashToken(token))
}

// Validate prüft ein Token. Ist die Session älter als einen Tag, wird sie
// gleitend verlängert und das neue Ablaufdatum zurückgegeben (sonst Zero-Time).
func (s *Service) Validate(ctx context.Context, token string) (renewed time.Time, err error) {
	if token == "" {
		return time.Time{}, store.ErrNotFound
	}
	h := hashToken(token)
	exp, err := s.store.AuthSessionExpiry(ctx, h)
	if err != nil {
		return time.Time{}, err
	}
	full := s.now().Add(s.sessionTTL)
	if full.Sub(exp) > 24*time.Hour {
		if err := s.store.ExtendAuthSession(ctx, h, full); err != nil {
			return time.Time{}, err
		}
		return full, nil
	}
	return time.Time{}, nil
}

func (s *Service) User() string { return s.user }

// ClientIP liefert die IP des Clients (X-Forwarded-For nur, wenn dem Proxy vertraut wird).
func (s *Service) ClientIP(r *http.Request) string {
	if s.trustProxy {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			return strings.TrimSpace(strings.Split(xff, ",")[0])
		}
		if xr := r.Header.Get("X-Real-IP"); xr != "" {
			return strings.TrimSpace(xr)
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// IsSecure gibt an, ob die Anfrage über HTTPS kam (für das Secure-Cookie-Flag).
func (s *Service) IsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return s.trustProxy && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

// SetCookie setzt das Session-Cookie.
func (s *Service) SetCookie(w http.ResponseWriter, r *http.Request, token string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   s.IsSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

func (s *Service) ClearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   s.IsSecure(r),
		SameSite: http.SameSiteStrictMode,
	})
}

// Middleware lässt nur Anfragen mit gültiger Session durch.
func (s *Service) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil {
			unauthorized(w)
			return
		}
		renewed, err := s.Validate(r.Context(), c.Value)
		if err != nil {
			unauthorized(w)
			return
		}
		if !renewed.IsZero() {
			s.SetCookie(w, r, c.Value, renewed)
		}
		next.ServeHTTP(w, r)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"unauthorized"}`))
}

// PurgeLoop entfernt regelmäßig abgelaufene Sessions und alte Limiter-Einträge.
func (s *Service) PurgeLoop(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		s.store.PurgeExpiredAuthSessions(ctx)
		s.limiter.gc(s.now())
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

// limiter zählt Fehlversuche pro IP in einem festen Zeitfenster.
type limiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]*limitEntry
}

type limitEntry struct {
	start    time.Time
	failures int
}

func newLimiter(max int, window time.Duration) *limiter {
	return &limiter{max: max, window: window, entries: map[string]*limitEntry{}}
}

func (l *limiter) allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok || now.Sub(e.start) >= l.window {
		return true
	}
	return e.failures < l.max
}

func (l *limiter) fail(ip string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.entries[ip]
	if !ok || now.Sub(e.start) >= l.window {
		l.entries[ip] = &limitEntry{start: now, failures: 1}
		return
	}
	e.failures++
}

func (l *limiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, ip)
}

func (l *limiter) gc(now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for ip, e := range l.entries {
		if now.Sub(e.start) >= l.window {
			delete(l.entries, ip)
		}
	}
}
