# Workout – Kettlebell-Trainings-App für den Homeserver

Selbst gehostete Web-App (Handy zuerst, Desktop geht auch) zum Zusammenstellen und Abspielen von Kettlebell-Workouts:

- **Workout-Builder** mit Blöcken: Sätze, Zirkel, Intervall/Tabata, EMOM, AMRAP, Pause. Gewichts-Schnellwahl für die eigenen Kettlebells (Standard 12 / 16 kg), einseitige Übungen werden automatisch links/rechts ausgeführt.
- **Player** mit automatischen Timern, 3‑2‑1-Tönen, deutschen Sprachansagen, Pause/Zurück/Überspringen, Bildschirm bleibt an.
- **Kamera-Spiegel**: Kamerabild als Hintergrund, Timer darüber. Sätze manuell oder automatisch (pro Block einstellbar) **aufnehmen** und später im **Review** mit Zeitlupe, Einzelbild-Schritten und Skelett-Overlay ansehen.
- **Formerkennung** (MediaPipe, läuft komplett lokal im Browser): Skelett-Overlay, Wiederholungen zählen und einfache Hinweise für Swing, Kniebeuge, Hinge/Kreuzheben und Überkopfdrücken.
- **Verlauf** mit Protokoll, bewegtem Volumen, Trainingskalender und Notizen.
- Hell-/Dunkelmodus, Login für einen Benutzer, alles auf Deutsch.

## Schnellstart mit Docker Compose

GitHub Actions testet jeden Push und baut bei Pushes auf `main` bzw. Tags `v*` das Image
`ghcr.io/jacolate/workoutbuddy` (amd64 + arm64). Auf dem Homeserver braucht es nur `docker-compose.yml` und `.env`:

```bash
# Nur bei privatem Repo: einmalig mit einem GitHub-Token (Scope read:packages) anmelden
docker login ghcr.io -u jacolate

cp .env.example .env
docker compose run --rm workout hash-password   # Passwort eingeben, Hash in .env eintragen
# In .env: AUTH_USER, AUTH_PASSWORD_HASH (in '…'), TLS_HOSTS (IP/Hostname des Servers)
docker compose up -d
```

Update auf die neueste Version: `docker compose pull && docker compose up -d`.
Eine feste Version statt `latest`: Tag `v1.0.0` pushen und in `docker-compose.yml` `…/workoutbuddy:1.0.0` eintragen.
Lokal bauen statt ziehen: in `docker-compose.yml` `image:` durch `build: .` ersetzen.

Dann am Handy **`https://<server-ip>:8443`** öffnen, die Zertifikatswarnung einmal bestätigen und anmelden. Tipp: Über „Zum Startbildschirm hinzufügen“ lässt sich die App wie eine native App starten.

Daten (SQLite-Datenbank, Aufnahmen, TLS-Zertifikat) liegen im Docker-Volume `workout-data`.

### Warum HTTPS?

Browser geben **Kamera, Wake-Lock (Bildschirm an) und Installation als App nur über HTTPS** frei (Ausnahme: `localhost`). Über `http://192.168.x.x` funktioniert der Kamera-Spiegel also nicht – auch nicht im VPN. Deshalb startet der Container standardmäßig mit `TLS_MODE=self-signed`: Beim ersten Start wird ein 10 Jahre gültiges Zertifikat erzeugt (für `localhost`, die IPs/Hosts aus `TLS_HOSTS` und die Container-IPs). Das Zertifikat bleibt im Volume, die Warnung muss also nur einmal bestätigt werden.

Wenn bereits ein HTTPS-Reverse-Proxy (Traefik, Caddy, nginx …) davor sitzt: in `.env` `TLS_MODE=off` und `TRUST_PROXY=true` setzen und den Proxy auf Port 8080 des Containers zeigen lassen.

## Konfiguration (Umgebungsvariablen)

| Variable | Standard | Bedeutung |
|---|---|---|
| `AUTH_USER` | – (Pflicht) | Benutzername |
| `AUTH_PASSWORD_HASH` | – (Pflicht*) | bcrypt-Hash, erzeugt mit `hash-password` |
| `AUTH_PASSWORD` | – | *Alternative im Klartext (wird beim Start gehasht, mit Warnung) |
| `TLS_MODE` | `off` (Compose: `self-signed`) | `off`, `self-signed` oder `files` |
| `TLS_HOSTS` | – | Zusätzliche IPs/Hostnamen fürs selbstsignierte Zertifikat (kommagetrennt) |
| `TLS_CERT` / `TLS_KEY` | – | Eigene Zertifikatsdateien (`TLS_MODE=files`) |
| `TRUST_PROXY` | `false` | `X-Forwarded-For/-Proto` auswerten (nur hinter eigenem Proxy!) |
| `SESSION_DAYS` | `30` | Wie lange man angemeldet bleibt (verlängert sich bei Nutzung) |
| `MAX_UPLOAD_MB` | `500` | Maximale Größe einer Aufnahme |
| `ADDR` | `:8080` | Listen-Adresse |
| `DATA_DIR` | `./data` (Docker: `/data`) | Datenverzeichnis |

Ohne Zugangsdaten startet der Server bewusst nicht.

### Login & Sicherheit

- Ein Benutzer, Zugangsdaten nur per Umgebungsvariable (kein „ersten Account anlegen“ im Browser, den jemand anderes im Netz zuerst nutzen könnte).
- Session-Cookie `HttpOnly`, `SameSite=Strict`, `Secure` bei HTTPS; das Token liegt in der DB nur als SHA-256-Hash. Abmelden macht es serverseitig ungültig.
- Alle API-Routen inklusive Videodateien erfordern Login. Fehlversuche sind auf 5 pro Minute und IP begrenzt.

## Kamera & Formerkennung – Tipps

- Für **Swing, Kniebeuge, Kreuzheben** das Handy **seitlich** in ca. 2–3 m Abstand aufstellen, für **Drücken frontal**. Der ganze Körper sollte im Bild sein.
- Welche Übung wie ausgewertet wird, stellst du unter **Übungen → Formerkennung** ein (Symbol 👁 in den Listen).
- Die Hinweise sind **grobe 2D-Heuristiken** (Gelenkwinkel), keine exakte Bewegungsanalyse. Die Schwellwerte stehen gesammelt in `frontend/src/lib/pose/profiles.ts` (`THRESHOLDS`) und lassen sich nach ein paar echten Trainings leicht anpassen.
- Im Review am besten mit 0,5× abspielen, dann zählt die Analyse am zuverlässigsten.
- Optional unter Einstellungen: „Automatisch weiter bei erkannten Reps“.

## Entwicklung

Voraussetzungen: Go ≥ 1.26, Node ≥ 22.17.

```bash
make dev    # Go-API auf :8080 + Vite auf http://localhost:5173 (Login: dev / devpassword)
make test   # Go-Tests, Vitest, svelte-check
make build  # bin/workout-app – einzelnes Binary mit eingebettetem Frontend
```

Aufbau:

```
backend/                 Go (chi, SQLite via modernc – ohne CGO)
  cmd/server/            main, Subcommands hash-password / healthcheck
  internal/api/          HTTP-Handler, SPA-Auslieferung
  internal/auth/         Login, Sessions, Rate-Limit
  internal/store/        SQLite, Migrationen, Seed (Übungen + Beispiel-Workouts)
  internal/media/        Speicherung der Aufnahmen
  internal/tlsutil/      Selbstsigniertes Zertifikat
  web/dist/              Frontend-Build (wird eingebettet)
frontend/                SvelteKit 3 (SPA) + Svelte 5 + Tailwind 4
  src/lib/timer/         Workout → Schrittfolge (compile.ts), Player-Engine
  src/lib/pose/          Geometrie, Rep-Zähler, Übungsprofile, MediaPipe
  src/lib/camera/        Kamera, MediaRecorder, Upload-Warteschlange
  src/routes/            Seiten (Training, Player, Verlauf, Review, Übungen, Einstellungen)
```

Das Workout-Format (Blöcke) ist in `frontend/src/lib/types.ts` definiert; das Backend speichert es unverändert als JSON.
