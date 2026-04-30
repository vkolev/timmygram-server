FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -o /out/timmygram ./cmd/server

# Runtime stage
FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        ffmpeg \
        sqlite3 \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /out/timmygram /app/timmygram
COPY templates /app/templates
COPY static /app/static
COPY migrations /app/migrations

RUN mkdir -p /app/videos /app/data

EXPOSE 8085

CMD ["/app/timmygram"]