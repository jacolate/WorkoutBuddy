package api

import (
	"errors"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler liefert den statischen Frontend-Build aus. Unbekannte Pfade bekommen
// index.html, damit das clientseitige Routing (z. B. /workouts/3) funktioniert.
func spaHandler(static fs.FS) http.Handler {
	fileServer := http.FileServerFS(static)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if p == "" {
			p = "index.html"
		}
		st, err := fs.Stat(static, p)
		if err != nil || st.IsDir() {
			if errors.Is(err, fs.ErrNotExist) && path.Ext(p) != "" && !strings.HasSuffix(p, ".html") {
				// Fehlende Assets nicht mit index.html beantworten.
				http.NotFound(w, r)
				return
			}
			serveIndex(w, r, static)
			return
		}
		switch {
		case strings.HasPrefix(p, "_app/immutable/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		case strings.HasPrefix(p, "mediapipe/"), strings.HasPrefix(p, "models/"):
			// Große Dateien (WASM, Modell): einen Tag cachen, danach revalidieren.
			w.Header().Set("Cache-Control", "public, max-age=86400")
		case p == "index.html" || strings.HasSuffix(p, ".webmanifest") || p == "service-worker.js":
			w.Header().Set("Cache-Control", "no-cache")
		}
		fileServer.ServeHTTP(w, r)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, static fs.FS) {
	b, err := fs.ReadFile(static, "index.html")
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Frontend ist nicht gebaut. Im Dev-Modus die App über den Vite-Server (Port 5173) öffnen."))
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(b)
}
