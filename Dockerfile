# Stage 1: Build the application
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o dkv main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/dkv /app/dkv

# Set sensible defaults
ENV DKV_SNAPSHOT_ENTRIES=100000
ENV DKV_COMPACTION_OVERHEAD=25000
ENV DKV_DATA_DIR="/app/data"
ENV DKV_TRANSPORT_ENDPOINT=":8080"
EXPOSE 8080

ENTRYPOINT ["/app/dkv"]
CMD ["serve"]