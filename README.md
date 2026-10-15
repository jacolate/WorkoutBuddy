# WorkoutBuddy

A self-hosted web app for kettlebell workouts at home. It is built for the phone first and also works on desktop.

- **Workout builder:** build a workout from blocks: sets, circuits, intervals/Tabata, EMOM, AMRAP and rest. You pick the weight with one tap from your own kettlebells (12 and 16 kg by default). Single-sided exercises automatically run for the left and right side.
- **Player:** timers run on their own, with 3-2-1 beeps and spoken announcements in German. You can pause, go back or skip, and the screen stays on during a workout.
- **Camera mirror:** your phone camera shows you as a mirror behind the timer. You can record sets by tapping a button, or have them recorded automatically for selected blocks. Afterwards you watch them in slow motion, frame by frame, with a skeleton overlay.
- **Form detection:** runs entirely in the browser with MediaPipe, so no video leaves your phone except the clips you upload to your own server. It draws a skeleton over you, counts reps and gives simple form hints for swings, squats, hinges/deadlifts and overhead presses.
- **History:** a log of every workout, the total weight you moved, a training calendar and notes.
- Light and dark mode, login for a single user. The interface is in German.

## Deployment (Docker Compose behind Caddy)

GitHub Actions tests every push. For each push to `main` and each `v*` tag it also publishes the image
`ghcr.io/jacolate/workoutbuddy`, for both `linux/amd64` and `linux/arm64`. Your Docker host only needs `docker-compose.yml` and a `.env` file.

```
Phone ──HTTPS──▶ Caddy (LXC) ──HTTP──▶ Docker host :8080 ──▶ workoutbuddy container
                 workout.jacolate.dev
```

### 1. Docker host

```bash
mkdir workoutbuddy && cd workoutbuddy
curl -fsSLO https://raw.githubusercontent.com/jacolate/WorkoutBuddy/main/docker-compose.yml
curl -fsSL  https://raw.githubusercontent.com/jacolate/WorkoutBuddy/main/.env.example -o .env

# Only needed if the repository/package is private (token scope: read:packages)
docker login ghcr.io -u jacolate

docker compose run --rm workoutbuddy hash-password   # paste the hash into .env (keep the single quotes)
# Also set AUTH_USER and TRUSTED_PROXIES (IP of the Caddy LXC) in .env
docker compose up -d
```

To update, run `docker compose pull && docker compose up -d`. To stay on a fixed version, push a tag such as `v1.0.0` and use `ghcr.io/jacolate/workoutbuddy:1.0.0` in the compose file.

The database, settings and recordings are stored in the Docker volume `workoutbuddy-data`.

### 2. Caddy

```caddyfile
workout.jacolate.dev {
	reverse_proxy <docker-host-ip>:8080
}
```

Caddy sets `X-Forwarded-For` and `X-Forwarded-Proto` by default. The app only trusts these headers when the request comes from an address listed in `TRUSTED_PROXIES`. It uses them to rate-limit logins per real client IP and to mark the session cookie as `Secure`.

### 3. DNS

Add a DNS rewrite in AdGuard Home: `workout.jacolate.dev` → the IP of the Caddy LXC.

### Why HTTPS matters

Browsers only allow **camera access, screen wake lock and installing the app on the home screen** over HTTPS (or on `localhost`). Always open the app via `https://workout.jacolate.dev`, not via `http://<docker-host-ip>:8080`.

Optionally, restrict port 8080 on the Docker host to the Caddy IP, for example with the Proxmox firewall. Then nobody can reach the app over plain HTTP.

## Configuration

All settings are environment variables in `.env`:

| Variable | Default | Description |
|---|---|---|
| `AUTH_USER` | – (required) | Login user name |
| `AUTH_PASSWORD_HASH` | – (required\*) | bcrypt hash from `hash-password` |
| `AUTH_PASSWORD` | – | \*Plain-text alternative; it is hashed at startup and the server logs a warning |
| `TRUSTED_PROXIES` | – | Comma-separated IPs/CIDRs of reverse proxies whose `X-Forwarded-*` headers are trusted |
| `WORKOUT_PORT` | `8080` | Host port that the container is published on (compose only) |
| `SESSION_DAYS` | `30` | How long a login lasts; each use extends it |
| `MAX_UPLOAD_MB` | `500` | Maximum size of a recorded clip |
| `ADDR` | `:8080` | Address the server listens on inside the container |
| `DATA_DIR` | `/data` | Data directory inside the container |

The server refuses to start without login credentials.

### Login & security

- **Single user:** credentials come only from the environment. There is no "create the first account in the browser" step that someone else on your network could get to first.
- **Session cookie:** the cookie is `HttpOnly` and `SameSite=Strict`, and `Secure` behind HTTPS. The database stores only a SHA-256 hash of the token, and logging out invalidates the session on the server.
- **What needs a login:** every API route, including the video files.
- **Brute-force protection:** failed logins are limited to 5 per minute per client IP.

## Camera & form detection tips

- **Camera position:** put the phone **to your side**, about 2–3 m away, for swings, squats and deadlifts. Put it **in front of you** for presses. Your whole body should be in the frame.
- **Per exercise:** choose which analysis an exercise uses under **Übungen → Formerkennung**.
- **Limits:** the hints are rough rules based on joint angles in the 2D camera image, not a precise movement analysis. All thresholds are in `THRESHOLDS` in `frontend/src/lib/pose/profiles.ts`, so you can tune them after a few real sessions.
- **Review:** reviewing a clip at 0.5× speed gives the most reliable rep counting.
- **Auto-advance:** under **Einstellungen**, you can let the app move on automatically once it has counted the target reps.

## Development

Requires Go ≥ 1.26 and Node ≥ 22.17.

```bash
make dev     # Go API on :8080 + Vite on http://localhost:5173 (login: dev / devpassword)
make test    # Go tests, Vitest, svelte-check
make build   # bin/workout-app – single binary with embedded frontend
make image   # build the container image locally
```

```
backend/                 Go (chi, SQLite via modernc – no CGO)
  cmd/server/            main, subcommands hash-password / healthcheck
  internal/api/          HTTP handlers, SPA serving
  internal/auth/         login, sessions, rate limit, trusted proxies
  internal/store/        SQLite, migrations, seed data (exercises + sample workouts)
  internal/media/        storage for recordings
  web/dist/              frontend build (embedded into the binary)
frontend/                SvelteKit 3 (SPA) + Svelte 5 + Tailwind 4
  src/lib/timer/         workout → step timeline (compile.ts), player engine
  src/lib/pose/          geometry, rep counter, exercise profiles, MediaPipe
  src/lib/camera/        camera, MediaRecorder, upload queue
  src/routes/            pages (training, player, history, review, exercises, settings)
.github/workflows/ci.yml tests + multi-arch image build → ghcr.io
```

The workout format (a list of blocks) is defined in `frontend/src/lib/types.ts`. The backend stores it unchanged as JSON.
