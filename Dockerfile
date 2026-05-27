# syntax=docker/dockerfile:1

FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /love-diary-go .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -H -u 10001 appuser

ENV TZ=Asia/Shanghai \
    GIN_MODE=release \
    PORT=3000

WORKDIR /app

COPY --from=builder /love-diary-go /app/love-diary-go

RUN mkdir -p /app/uploads && chown -R appuser:appuser /app/uploads

USER appuser

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:3000/health || exit 1

ENTRYPOINT ["/app/love-diary-go"]
