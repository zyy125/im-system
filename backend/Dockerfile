FROM golang:1.25 AS builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./backend/
WORKDIR /src/backend
RUN go mod download

COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/im-backend ./cmd

FROM debian:bookworm-slim

RUN useradd --system --create-home --home-dir /app appuser
RUN apt-get update \
    && apt-get install -y --no-install-recommends wget ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/im-backend /app/im-backend
COPY backend/config /app/config

RUN mkdir -p /app/storage/avatars && chown -R appuser:appuser /app

USER appuser

EXPOSE 8080 9090

CMD ["/app/im-backend"]
