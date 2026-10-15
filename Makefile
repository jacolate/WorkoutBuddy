.PHONY: dev backend-dev frontend-dev test build image hash-password

# Local development: Go API on :8080, Vite with hot reload on :5173 (proxies /api → :8080).
# Open http://localhost:5173 and log in with dev / devpassword (localhost allows the camera without HTTPS).
dev:
	$(MAKE) -j2 backend-dev frontend-dev

backend-dev:
	cd backend && ADDR=127.0.0.1:8080 DATA_DIR=../data AUTH_USER=dev AUTH_PASSWORD=devpassword go run ./cmd/server

frontend-dev:
	cd frontend && npm install && npm run assets && npm run dev

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npm test && npm run check

# Single binary with the embedded frontend → bin/workout-app
build:
	cd frontend && npm ci && npm run build
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/workout-app ./cmd/server

# Build the container image locally (CI publishes ghcr.io/jacolate/workoutbuddy)
image:
	docker build -t workoutbuddy:local .

hash-password:
	cd backend && go run ./cmd/server hash-password
