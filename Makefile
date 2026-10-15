.PHONY: dev backend-dev frontend-dev test build docker hash-password

# Lokale Entwicklung: Go-API auf :8080, Vite mit Hot-Reload auf :5173 (proxy /api → :8080)
# Login: dev / devpassword – im Browser http://localhost:5173 öffnen (localhost erlaubt Kamera auch ohne HTTPS)
dev:
	$(MAKE) -j2 backend-dev frontend-dev

backend-dev:
	cd backend && ADDR=127.0.0.1:8080 DATA_DIR=../data AUTH_USER=dev AUTH_PASSWORD=devpassword go run ./cmd/server

frontend-dev:
	cd frontend && npm install && npm run assets && npm run dev

test:
	cd backend && go vet ./... && go test ./...
	cd frontend && npm test && npm run check

# Einzelnes Binary mit eingebettetem Frontend → bin/workout-app
build:
	cd frontend && npm ci && npm run build
	cd backend && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../bin/workout-app ./cmd/server

docker:
	docker compose build

hash-password:
	cd backend && go run ./cmd/server hash-password
