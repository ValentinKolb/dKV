# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o dkv main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

RUN adduser -D -h /app appuser # Non-root user to run the application
WORKDIR /app

COPY --from=builder /app/dkv /app/dkv

RUN chown -R appuser:appuser /app

USER appuser


# Set sensible defaults
ENV DKV_SNAPSHOT_ENTRIES=5000
ENV DKV_COMPACTION_OVERHEAD=2500
ENV DKV_DATA_DIR="/app/data"
ENV DKV_ENDPOINT=":8080"
EXPOSE 8080

ENTRYPOINT ["/app/dkv", "serve"]