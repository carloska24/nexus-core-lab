# syntax=docker/dockerfile:1

FROM node:22-bookworm-slim AS frontend
WORKDIR /src/web/reference-version
COPY web/reference-version/package.json web/reference-version/package-lock.json ./
RUN npm ci
COPY web/reference-version/ ./
RUN npm run test:map && npm run build

FROM golang:1.27-bookworm AS backend
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/nexus-core-lab ./cmd/api

FROM gcr.io/distroless/static-debian12:nonroot AS runtime
WORKDIR /app
COPY --from=backend --chown=nonroot:nonroot /out/nexus-core-lab /app/nexus-core-lab
COPY --from=frontend --chown=nonroot:nonroot /src/web/reference-version/dist /app/web
ENV PORT=8080 \
    NEXUS_WEB_DIR=/app/web \
    PUBLIC_DEMO_MODE=true
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/nexus-core-lab"]
