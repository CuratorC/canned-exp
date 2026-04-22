# Stage 1: Build
FROM golang:1.26-alpine AS builder
RUN apk add --no-cache git

WORKDIR /build

# 先复制 go.mod/go.sum，利用 Docker 层缓存
COPY go.mod go.sum ./
RUN go mod download

# 复制源码并编译
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o canned-exp ./main.go

# Stage 2: Runtime
FROM alpine:3.21
RUN apk add --no-cache tzdata ca-certificates

WORKDIR /app
COPY --from=builder /build/canned-exp .
COPY <<'EOF' entrypoint.sh
#!/bin/sh
./canned-exp migrate run
exec ./canned-exp serve
EOF
RUN chmod +x entrypoint.sh && mkdir -p storage

EXPOSE 3100

ENTRYPOINT ["./entrypoint.sh"]
