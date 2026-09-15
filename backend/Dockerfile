# syntax=docker/dockerfile:1

FROM golang:1.27-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/worker ./cmd/worker \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/seed ./cmd/seed

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 hookforge
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/worker /app/worker
COPY --from=builder /out/seed /app/seed
USER hookforge
EXPOSE 8080
CMD ["/app/api"]
