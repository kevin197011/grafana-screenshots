# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY cmd/ cmd/
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags="-s -w" \
    -o /out/grafana-screenshot ./cmd/grafana-screenshot

# 在 builder 阶段安装 tzdata（CI 可访问 apk）；运行镜像不执行 apk，避免 Rocky 等环境构建卡住
RUN apk add --no-cache tzdata

# 运行镜像不执行 apk（Rocky/CentOS 等环境访问 dl-cdn.alpinelinux.org 易卡住）
FROM alpine:3.20

COPY --from=builder /etc/ssl/certs /etc/ssl/certs
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

WORKDIR /app
COPY --from=builder /out/grafana-screenshot /app/grafana-screenshot
COPY docker/entrypoint.sh /entrypoint.sh

RUN adduser -D -u 1000 app \
    && chmod +x /entrypoint.sh \
    && mkdir -p /data/screenshots \
    && chown -R app:app /app /data

# entrypoint 以 root 修正挂载卷权限后降权为 app
ENV TZ=Asia/Hong_Kong
ENTRYPOINT ["/entrypoint.sh"]
EXPOSE 8111
CMD ["server"]
