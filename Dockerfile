# Stage 1: Builder
FROM golang:1.25-alpine3.23 AS builder

WORKDIR /app

# Install libvips for CGO compilation (golang:latest is Debian-based)
RUN apk add --no-cache \
  vips-dev \
  pkgconfig \
  build-base

COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN make build-linux

# Stage 2: Final Image
FROM alpine:3.23

# Install only the runtime vips library (not dev tools)
RUN apk add --no-cache \
    vips \
    ca-certificates

COPY --from=builder /app/bin/yoollive-api-server ./yoollive-api-server
RUN chmod +x ./yoollive-api-server

ENTRYPOINT ["./yoollive-api-server"]