# Grafana Screenshot

Agentless Grafana Dashboard 定时截图：通过 `/render` API 生成 PNG，并推送到 Lark 群。

## 目录结构

```
grafana-screenshot/
├── cmd/grafana-screenshot/   # Go 源码（server + once + scheduler）
├── data/
│   └── screenshots/        # 截图输出（挂载卷）
├── docker/
│   ├── Dockerfile
│   └── entrypoint.sh
├── docker-compose.yml
├── grafana.env.example
└── Makefile
```

## 快速开始

```bash
cp grafana.env.example grafana.env
# 编辑 grafana.env，填入 GRAFANA_TOKEN、GRAFANA_URL 和 Lark 配置
# 若已有 .env：mv .env grafana.env，并删除或改名根目录下的 .env（Compose 会自动读 .env 并误解析 $__all）

make build       # 拉取 ghcr.io/kevin197011/grafana-screenshots:latest
make up          # 启动容器，HTTP :8111
make logs        # 查看容器标准输出（docker compose logs）
make trigger     # GET 触发截图并发送到 Lark（可能需数分钟，见下方说明）
```

## HTTP 触发（默认模式）

容器默认以 `server` 启动，映射 **8111** 端口：

| 路径 | 说明 |
|------|------|
| `GET /health` | 健康检查，返回 `ok` |
| `GET /trigger` | 截图并发送到 Lark |
| `GET /trigger?dry-run=1` | 只截图，不发送 |

```bash
curl http://127.0.0.1:8111/health
curl --max-time 600 http://127.0.0.1:8111/trigger
```

全页 Dashboard 渲染较慢，建议 `curl` 加 `--max-time 600`（或更长）。若配置了 `TRIGGER_SECRET`，需加 `?token=你的密钥`。

仍要每日定时任务时，在 `grafana.env` 设 `ENABLE_SCHEDULER=true`（与 HTTP 服务同时运行）。仅定时、不要 HTTP 时：`docker compose run` 或改 `command: ["scheduler"]`。

## 定时配置

在 `grafana.env` 中设置（按 `TZ` 时区，默认 `Asia/Hong_Kong`）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `SCHEDULE_HOUR` | 小时 0–23 | `17` |
| `SCHEDULE_MINUTE` | 分钟 0–59 | `0` |
| `SCHEDULE_SECOND` | 秒 0–59 | `0` |
| `TZ` | 时区 | `Asia/Hong_Kong` |
| `RUN_ON_START` | 启动时立即执行一次 | `false` |

示例：每天 **17:30:15** 执行：

```env
SCHEDULE_HOUR=17
SCHEDULE_MINUTE=30
SCHEDULE_SECOND=15
```

## 常用命令

```bash
make trigger     # HTTP 触发截图并发送（容器须已 up）
make trigger-dry # HTTP 触发，仅截图
make dry-run     # 一次性容器执行，不发送
make once        # 一次性容器执行并发送
make down        # 停止容器
```

手动执行：

```bash
docker compose run --rm grafana-screenshot once --dry-run
```

## 配置说明

- **GRAFANA_TOKEN**：Service Account Token（`glsa_` 开头）
- **GRAFANA_URL**：Dashboard 完整 URL（写在 `grafana.env`；可含 `$__all` 等 Grafana 变量，勿将配置文件命名为 `.env`）
- **RENDER_TIMEOUT**：服务端等待面板查询秒数（默认 120）
- **RENDER_KIOSK**：截图时隐藏左侧菜单，默认 `true`；`tv` 进一步隐藏顶栏控件；`false` 保留完整 UI
- **RENDER_FULL_PAGE**：默认 `true`，使用 `height=-1` 滚动截取整页（避免只拍到首屏 1080px）；仅要视口大小时设 `false` 并指定 `RENDER_HEIGHT`
- **SCREENSHOT_RETENTION_DAYS**：`data/screenshots` 中截图保留天数，每次截图后自动删除过期文件（默认 30）

### Lark 群推送（唯一通知渠道）

在 [Lark 开放平台](https://open.larksuite.com/app) 创建企业自建应用，开启机器人能力并加入目标群，在 `grafana.env` 配置：

| 变量 | 说明 |
|------|------|
| `LARK_APP_ID` | 应用 App ID |
| `LARK_APP_SECRET` | 应用 App Secret |
| `LARK_CHAT_ID` | 群聊 ID（`oc_` 开头，可在 API 调试台「获取群列表」查看） |
| `LARK_API_BASE` | 可选，默认 `https://open.larksuite.com` |
| `LARK_HTTP_TIMEOUT` | Lark API 请求超时秒数（默认 120） |

所需权限（权限管理 → 开通）：`im:resource`（上传图片）、`im:message` / `im:message:send_as_bot`（发消息）。保存后需**发布应用版本**并确保机器人在群内。

日志仅输出到标准输出/标准错误，不写入项目内日志文件；用 `make logs` 或 `docker compose logs -f` 查看。

### 服务器上「卡住」的常见原因

| 现象 | 原因 | 处理 |
|------|------|------|
| `make up` 后无新输出 | **正常**：定时模式在等下次执行时间，日志会显示「下次执行: …（X 后）」 | 立即跑一次用 `make once`；或设 `RUN_ON_START=true` |
| 停在「正在请求 Grafana 渲染」很久 | 全页 `height=-1` + 面板多，渲染常需 **2～5 分钟**（最长约 `RENDER_TIMEOUT+30` 秒） | 加大 `RENDER_TIMEOUT=300`；或 `RENDER_FULL_PAGE=false` 只截首屏 |
| 停在「正在发送到 Lark」 | 服务器访问不了 `open.larksuite.com`（防火墙/无公网） | 放行出站 HTTPS |
| 一直无响应 | HTTP 连接挂死 | 调大 `LARK_HTTP_TIMEOUT`（默认 120 秒） |
| 渲染失败 | 服务器访问不了 `GRAFANA_URL` 域名（仅内网/VPN 可达） | 在服务器上 `curl -I` 测 Grafana；或改用内网地址 |
| `curl: Connection refused` 8111 | 容器未运行、未映射端口，或仍是旧镜像 `scheduler` 模式 | `make status`；`docker compose up -d --force-recreate`；`docker compose logs` 应出现 `HTTP 监听` |
| `permission denied` 写截图 | 宿主机 `data/screenshots` 属主不是 uid 1000 | `mkdir -p data/screenshots && chown -R 1000:1000 data/screenshots`；或拉最新镜像（entrypoint 会自动 chown） |

排查：另开终端 `docker compose logs -f`，看最后一行停在哪一步。

### Rocky Linux 9 上 `docker compose build` 卡住

旧版 Dockerfile 在运行镜像里执行 `apk add`，需访问 `dl-cdn.alpinelinux.org`，内网/国内机器常长时间无响应。当前镜像已改为从 builder 拷贝 CA 证书和时区，**不再 apk install**。

若卡在 `go mod download` / `go build`，多为拉取 Go 模块慢，可指定代理后重建：

```bash
export GOPROXY=https://goproxy.cn,direct
docker compose build --progress=plain
docker compose up -d
```

需 BuildKit（Docker 23+ 默认开启）。仍慢时加 `DOCKER_BUILDKIT=1` 并查看 `--progress=plain` 卡在哪一步。

## CI 镜像（GitHub Actions）

推送 `main` 或 `v*` 标签后，在 [Packages](https://github.com/kevin197011/grafana-screenshots/pkgs/container/grafana-screenshots) 发布 **仅 `linux/amd64`** 镜像：

```bash
docker pull ghcr.io/kevin197011/grafana-screenshots:latest
```

`pull_request` 只构建校验，不推送。

## 要求

- Docker / Docker Compose
- Grafana 已启用 **image renderer** 插件
- Token 对目标 Dashboard 有查看权限
