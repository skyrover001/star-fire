# syntax=docker/dockerfile:1
FROM golang:1.20-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o server ./cmd/server
FROM alpine:3.17
RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Shanghai
RUN adduser -D -g '' appuser
WORKDIR /app
COPY --from=builder /app/server .
RUN mkdir -p /app/configs && chown -R appuser:appuser /app
USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./server"]