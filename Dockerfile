# ---- Stage 1: Build Frontend ----
FROM node:22-alpine AS frontend-builder

WORKDIR /app/web/frontend
COPY web/frontend/package.json web/frontend/package-lock.json ./
RUN npm ci --ignore-scripts
COPY web/frontend/ ./
RUN npm run build

# ---- Stage 2: Build Go Binary ----
FROM golang:1.22-alpine AS go-builder

RUN apk add --no-cache gcc musl-dev git

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .

# Copy frontend build output into the expected location
COPY --from=frontend-builder /app/web/static/dist ./web/static/dist

RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w -X main.Version=0.1.0 -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o /app/bin/aetherdev ./cmd/aetherdev

# ---- Stage 3: Production Image ----
FROM alpine:3.20

RUN apk add --no-cache git ca-certificates tzdata

RUN adduser -D -h /home/aetherdev aetherdev

WORKDIR /home/aetherdev

# Copy binary
COPY --from=go-builder /app/bin/aetherdev /usr/local/bin/aetherdev

# Copy web assets (SPA build + static CSS/images)
COPY --from=go-builder /app/web/static ./web/static
COPY --from=go-builder /app/web/templates ./web/templates

RUN mkdir -p /home/aetherdev/data && chown -R aetherdev:aetherdev /home/aetherdev

USER aetherdev

EXPOSE 3000

VOLUME ["/home/aetherdev/data"]

ENTRYPOINT ["aetherdev"]
CMD ["--data", "/home/aetherdev/data"]
