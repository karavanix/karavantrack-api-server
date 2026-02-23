# Stage 1: Builder
FROM golang:latest AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN make build-linux

# Stage 2: Final Image
FROM alpine:latest

COPY --from=builder /app/bin/karavantruck-api-server ./karavantruck-api-server
RUN chmod +x ./karavantruck-api-server

ENTRYPOINT ["./karavantruck-api-server"]