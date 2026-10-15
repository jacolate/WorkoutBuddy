// Package api stellt die HTTP-Schnittstelle bereit.
package api

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"workoutapp/internal/auth"
	"workoutapp/internal/media"
	"workoutapp/internal/store"
)

type Server struct {
	store          *store.Store
	media          *media.Storage
	auth           *auth.Service
	maxUploadBytes int64
}

type Options struct {
	Store          *store.Store
	Media          *media.Storage
	Auth           *auth.Service
	MaxUploadBytes int64
	// Static ist der Frontend-Build (index.html an der Wurzel); nil deaktiviert die Auslieferung.
	Static fs.FS
}

// New baut den kompletten HTTP-Handler (API + Frontend).
func New(o Options) http.Handler {
	s := &Server{store: o.Store, media: o.Media, auth: o.Auth, maxUploadBytes: o.MaxUploadBytes}

	r := chi.NewRouter()
	// Kein middleware.RealIP: X-Forwarded-For wird nur von TRUSTED_PROXIES ausgewertet (auth.ClientIP),
	// sonst könnte man das Login-Rate-Limit mit gefälschten Headern umgehen.
	r.Use(middleware.Recoverer)
	r.Use(securityHeaders)

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.NoCache)
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		})
		r.Post("/auth/login", s.login)
		r.Post("/auth/logout", s.logout)

		r.Group(func(r chi.Router) {
			r.Use(s.auth.Middleware)
			r.Get("/auth/me", s.me)

			r.Get("/exercises", s.listExercises)
			r.Post("/exercises", s.createExercise)
			r.Get("/exercises/{id}", s.getExercise)
			r.Put("/exercises/{id}", s.updateExercise)
			r.Delete("/exercises/{id}", s.deleteExercise)

			r.Get("/workouts", s.listWorkouts)
			r.Post("/workouts", s.createWorkout)
			r.Get("/workouts/{id}", s.getWorkout)
			r.Put("/workouts/{id}", s.updateWorkout)
			r.Delete("/workouts/{id}", s.deleteWorkout)
			r.Post("/workouts/{id}/duplicate", s.duplicateWorkout)

			r.Get("/sessions", s.listSessions)
			r.Post("/sessions", s.createSession)
			r.Get("/sessions/{id}", s.getSession)
			r.Patch("/sessions/{id}", s.patchSession)
			r.Delete("/sessions/{id}", s.deleteSession)

			r.Get("/recordings", s.listRecordings)
			r.Post("/recordings", s.uploadRecording)
			r.Get("/recordings/{id}", s.getRecording)
			r.Patch("/recordings/{id}", s.patchRecording)
			r.Delete("/recordings/{id}", s.deleteRecording)
			r.Get("/recordings/{id}/file", s.recordingFile)

			r.Get("/settings", s.getSettings)
			r.Put("/settings", s.putSettings)
		})

		r.NotFound(func(w http.ResponseWriter, r *http.Request) { writeError(w, http.StatusNotFound, "not found") })
	})

	if o.Static != nil {
		r.With(middleware.Compress(5, "text/html", "text/css", "text/javascript", "application/javascript",
			"application/json", "application/manifest+json", "image/svg+xml", "application/wasm")).
			Handle("/*", spaHandler(o.Static))
	}
	return r
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "same-origin")
		h.Set("Permissions-Policy", "camera=(self), microphone=(), geolocation=()")
		next.ServeHTTP(w, r)
	})
}

// --- Helfer ---

const maxJSONBody = 2 << 20

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Warn("write json", "err", err)
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json: "+err.Error())
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

// handleErr übersetzt Store-Fehler in HTTP-Antworten.
func handleErr(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	slog.Error("request failed", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

// --- Auth ---

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	ip := s.auth.ClientIP(r)
	token, expires, err := s.auth.Login(r.Context(), ip, req.Username, req.Password)
	switch {
	case errors.Is(err, auth.ErrRateLimited):
		w.Header().Set("Retry-After", "60")
		writeError(w, http.StatusTooManyRequests, "Zu viele Versuche – bitte eine Minute warten.")
		return
	case errors.Is(err, auth.ErrInvalidCredentials):
		slog.Warn("failed login", "ip", ip)
		writeError(w, http.StatusUnauthorized, "Benutzername oder Passwort falsch.")
		return
	case err != nil:
		handleErr(w, err)
		return
	}
	s.auth.SetCookie(w, r, token, expires)
	writeJSON(w, http.StatusOK, map[string]any{"user": s.auth.User(), "expiresAt": expires.UTC().Format(time.RFC3339)})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(auth.CookieName); err == nil {
		if err := s.auth.Logout(r.Context(), c.Value); err != nil {
			handleErr(w, err)
			return
		}
	}
	s.auth.ClearCookie(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"user": s.auth.User()})
}
