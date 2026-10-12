# syntax=docker/dockerfile:1.7

# --- development: hot reload via Air ---
FROM golang:1.23.6-alpine AS development

RUN apk add --no-cache git ca-certificates curl build-base && \
    go install github.com/air-verse/air@v1.61.7

WORKDIR /app

COPY go.mod ./
COPY go.sum* ./
RUN go mod download

COPY . .
RUN go mod tidy

EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

# --- production build (for future use; not used by local Compose) ---
FROM golang:1.23.6-alpine AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server

FROM alpine:3.21 AS production
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/server /app/server
EXPOSE 8080
USER nobody
ENTRYPOINT ["/app/server"]
