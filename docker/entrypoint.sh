#!/bin/sh
set -eu

screenshot_dir="${SCREENSHOT_DIR:-/data/screenshots}"
mkdir -p "$screenshot_dir"

# 宿主机 bind mount 常为 root 创建；compose 使用 user: "0:0" 时直接以 root 运行
if [ "$(id -u)" = "0" ]; then
	chown -R root:root "$screenshot_dir" 2>/dev/null || chmod -R a+rwX "$screenshot_dir"
	exec /app/grafana-screenshot "$@"
fi

exec /app/grafana-screenshot "$@"
