# syntax=docker/dockerfile:1.7

# ============================================================
# Stage 1: build
# 使用 golang:alpine 编译,产物为静态二进制 (CGO_ENABLED=0)
# ============================================================
FROM golang:1.26-alpine AS builder

WORKDIR /src

# 先拷贝 go.mod / go.sum 单独缓存依赖层,源码改动不会失效
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# 拷贝其余源码
COPY . .

# 与 .github/workflows/release.yml 中的编译参数保持一致
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0 \
    GOAMD64=v1
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-s -w -buildid=" \
        -o /out/telegram-drive-bot \
        ./cmd/bot

# ============================================================
# Stage 2: runtime
# alpine 提供 ca-certificates / tzdata
# ============================================================
FROM alpine:3.20

# ca-certificates: 访问 Telegram API / Supabase 等 HTTPS 服务必需
# tzdata:          日志时间戳与数据库时区处理
# 固定 UID/GID, 方便用户在 host 端管理挂载目录的权限 (chown 10001:10001 ./data)
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 app \
    && adduser  -S -u 10001 -G app app

WORKDIR /app

# 仅复制二进制 + 示例配置,避免源码进入运行镜像
COPY --from=builder /out/telegram-drive-bot /app/telegram-drive-bot
COPY --chown=app:app .env.example /app/.env.example

# 运行时数据目录, 目前没有东西要放到data里, 先写了为以后其他功能留着(比如日志之类的)
RUN mkdir -p /app/data && chown -R app:app /app

USER app

# 应用使用 long polling,无需暴露端口
ENTRYPOINT ["/app/telegram-drive-bot"]
