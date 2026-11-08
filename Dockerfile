# syntax=docker/dockerfile:1

# ---------- Etapa de compilación ----------
FROM golang:1.22-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/gastos-api ./cmd/api

# ---------- Imagen final ----------
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata wget \
    && addgroup -S app \
    && adduser -S -G app app \
    && mkdir -p /data \
    && chown -R app:app /data

WORKDIR /app

COPY --from=builder /out/gastos-api /app/gastos-api

USER app

ENV GIN_MODE=release \
    PORT=8080 \
    DB_PATH=/data/gastos.db

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/health >/dev/null || exit 1

CMD ["/app/gastos-api"]
