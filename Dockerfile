# Stage 1: Builder
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN make build-linux

# Stage 2: Final Image
FROM alpine:latest

COPY --from=builder /app/bin/yoollive-api-server ./yoollive-api-server
RUN chmod +x ./yoollive-api-server

ENTRYPOINT ["./yoollive-api-server"]