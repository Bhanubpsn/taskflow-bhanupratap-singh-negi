# ── Build stage ──────────────────────────────────────────────────────────────
FROM golang:alpine AS builder

WORKDIR /build

# Cache dependencies as a separate layer
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o api .

# ── Runtime stage ─────────────────────────────────────────────────────────────
FROM alpine:latest

# ca-certificates: needed for TLS; tzdata: correct timestamps in logs
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /build/api        ./api
COPY --from=builder /build/migrations ./migrations

EXPOSE 3000

CMD ["./api"]
