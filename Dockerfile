# ── Build ──
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 make server

# ── Runtime ──
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -u 1000 entropy
USER entropy

WORKDIR /app

COPY --from=builder /app/bin/entropy-server /usr/local/bin/entropy-server
COPY --from=builder /app/configs /app/configs

EXPOSE 443 8443

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:9876/api/health || exit 1

ENTRYPOINT ["entropy-server"]
CMD ["serve", "-c", "/app/configs/server-example.yaml"]
