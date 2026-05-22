# Graph Report - grafana-screenshot  (2026-05-22)

## Corpus Check
- 6 files · ~22,269 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 68 nodes · 129 edges · 11 communities (10 shown, 1 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `181b1ae8`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]

## God Nodes (most connected - your core abstractions)
1. `runOnce()` - 10 edges
2. `Grafana Screenshot` - 9 edges
3. `runServer()` - 8 edges
4. `envInt()` - 8 edges
5. `runScheduler()` - 7 edges
6. `envOr()` - 7 edges
7. `renderTimeoutSec()` - 7 edges
8. `sendLark()` - 7 edges
9. `buildRenderURL()` - 6 edges
10. `main()` - 5 edges

## Surprising Connections (you probably didn't know these)
- `runServer()` --calls--> `envOr()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 1 → community 3_
- `runScheduler()` --calls--> `envInt()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 1 → community 7_
- `runOnce()` --calls--> `renderTimeoutSec()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 1 → community 4_
- `larkAPIBase()` --calls--> `envOr()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 3 → community 2_
- `renderHeight()` --calls--> `envInt()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 7 → community 4_

## Communities (11 total, 1 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.12
Nodes (15): CI 镜像（GitHub Actions）, code:block1 (grafana-screenshot/), code:bash (cp grafana.env.example grafana.env), code:bash (curl http://127.0.0.1:8111/health), code:env (SCHEDULE_HOUR=17), code:bash (make trigger     # HTTP 触发截图并发送（容器须已 up）), code:bash (docker compose run --rm grafana-screenshot once --dry-run), code:bash (docker pull ghcr.io/kevin197011/grafana-screenshots:latest) (+7 more)

### Community 1 - "Community 1"
Cohesion: 0.31
Nodes (11): envBool(), fatal(), handleTrigger(), loadEnv(), main(), restoreGrafanaURLFromFile(), runOnce(), runOnceFromFlags() (+3 more)

### Community 2 - "Community 2"
Cohesion: 0.39
Nodes (9): larkAPIBase(), larkHTTPClient(), larkSendImage(), larkSendMessage(), larkSendText(), larkTenantToken(), larkUploadImage(), sendLark() (+1 more)

### Community 3 - "Community 3"
Cohesion: 0.42
Nodes (8): applyKioskQuery(), deliver(), envOr(), grafanaTimeRangeLabel(), larkCaption(), larkConfig(), parseGrafanaTimeExpr(), schedulerLocation()

### Community 4 - "Community 4"
Cohesion: 0.53
Nodes (6): buildRenderURL(), captureDashboard(), envRaw(), renderHeight(), renderTimeoutSec(), renderWidth()

### Community 5 - "Community 5"
Cohesion: 0.40
Nodes (5): code:bash (export GOPROXY=https://goproxy.cn,direct), Lark 群推送（唯一通知渠道）, Rocky Linux 9 上 `docker compose build` 卡住, 服务器上「卡住」的常见原因, 配置说明

### Community 6 - "Community 6"
Cohesion: 0.50
Nodes (3): go.toolsManagement.checkForUpdates, gopls, build.directoryFilters

### Community 7 - "Community 7"
Cohesion: 0.67
Nodes (3): envInt(), envIntAlias(), pruneOldScreenshots()

## Knowledge Gaps
- **15 isolated node(s):** `go.toolsManagement.checkForUpdates`, `build.directoryFilters`, `version`, `configurations`, `code:block1 (grafana-screenshot/)` (+10 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Grafana Screenshot` connect `Community 0` to `Community 5`?**
  _High betweenness centrality (0.078) - this node is a cross-community bridge._
- **Why does `配置说明` connect `Community 5` to `Community 0`?**
  _High betweenness centrality (0.031) - this node is a cross-community bridge._
- **What connects `go.toolsManagement.checkForUpdates`, `build.directoryFilters`, `version` to the rest of the system?**
  _15 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.125 - nodes in this community are weakly interconnected._