# syntax=docker/dockerfile:1

# ---- Frontend (SvelteKit → statische Dateien, plattformunabhängig) ----
FROM --platform=$BUILDPLATFORM node:24-alpine AS web
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
# Baut nach ../backend/web/dist; lädt dabei MediaPipe-WASM + Pose-Modell (falls nicht vorhanden)
RUN mkdir -p ../backend/web && npm run build

# ---- Backend (Go, Frontend eingebettet; Cross-Compile für die Zielplattform) ----
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS server
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=web /src/backend/web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /out/workout-app ./cmd/server \
	&& mkdir -p /out/data

# ---- Laufzeit ----
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=server /out/workout-app /app/workout-app
# /data gehört dem nonroot-User, damit ein benanntes Volume beschreibbar ist
COPY --from=server --chown=65532:65532 /out/data /data
ENV ADDR=:8080 \
	DATA_DIR=/data
VOLUME /data
EXPOSE 8080
USER nonroot:nonroot
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s CMD ["/app/workout-app", "healthcheck"]
ENTRYPOINT ["/app/workout-app"]
