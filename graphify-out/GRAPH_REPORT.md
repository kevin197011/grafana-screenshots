# Graph Report - grafana-screenshot  (2026-05-25)

## Corpus Check
- 6 files · ~22,411 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 72 nodes · 142 edges · 14 communities (9 shown, 5 thin omitted)
- Extraction: 100% EXTRACTED · 0% INFERRED · 0% AMBIGUOUS
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `65d3584c`
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
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]

## God Nodes (most connected - your core abstractions)
1. `runScheduler()` - 12 edges
2. `runOnce()` - 11 edges
3. `runServer()` - 9 edges
4. `envInt()` - 9 edges
5. `Grafana Screenshot` - 9 edges
6. `envOr()` - 7 edges
7. `renderTimeoutSec()` - 7 edges
8. `sendLark()` - 7 edges
9. `envRaw()` - 6 edges
10. `buildRenderURL()` - 6 edges

## Surprising Connections (you probably didn't know these)
- `loadEnv()` --calls--> `restoreGrafanaURLFromFile()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 1 → community 3_
- `larkAPIBase()` --calls--> `envOr()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 1 → community 2_
- `larkHTTPClient()` --calls--> `envInt()`  [EXTRACTED]
  cmd/grafana-screenshot/main.go → cmd/grafana-screenshot/main.go  _Bridges community 3 → community 2_

## Communities (14 total, 5 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.40
Nodes (4): code:bash (cp grafana.env.example grafana.env), Grafana Screenshot, 快速开始, 要求

### Community 1 - "Community 1"
Cohesion: 0.25
Nodes (14): cronStdlibLogger, deliver(), envBool(), envOr(), fatal(), handleTrigger(), larkConfig(), loadEnv() (+6 more)

### Community 2 - "Community 2"
Cohesion: 0.39
Nodes (9): larkAPIBase(), larkHTTPClient(), larkSendImage(), larkSendMessage(), larkSendText(), larkTenantToken(), larkUploadImage(), sendLark() (+1 more)

### Community 3 - "Community 3"
Cohesion: 0.25
Nodes (17): applyKioskQuery(), buildRenderURL(), captureDashboard(), envInt(), envIntAlias(), envRaw(), grafanaTimeRangeLabel(), larkCaption() (+9 more)

### Community 4 - "Community 4"
Cohesion: 0.67
Nodes (3): code:bash (make trigger     # HTTP 触发截图并发送（容器须已 up）), code:bash (docker compose run --rm grafana-screenshot once --dry-run), 常用命令

### Community 5 - "Community 5"
Cohesion: 0.40
Nodes (5): code:bash (export GOPROXY=https://goproxy.cn,direct), Lark 群推送（唯一通知渠道）, Rocky Linux 9 上 `docker compose build` 卡住, 服务器上「卡住」的常见原因, 配置说明

### Community 6 - "Community 6"
Cohesion: 0.50
Nodes (3): go.toolsManagement.checkForUpdates, gopls, build.directoryFilters

## Knowledge Gaps
- **15 isolated node(s):** `go.toolsManagement.checkForUpdates`, `build.directoryFilters`, `version`, `configurations`, `code:block1 (grafana-screenshot/)` (+10 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `Grafana Screenshot` connect `Community 0` to `Community 4`, `Community 5`, `Community 7`, `Community 11`, `Community 12`, `Community 13`?**
  _High betweenness centrality (0.069) - this node is a cross-community bridge._
- **Why does `配置说明` connect `Community 5` to `Community 0`?**
  _High betweenness centrality (0.028) - this node is a cross-community bridge._
- **Why does `常用命令` connect `Community 4` to `Community 0`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **What connects `go.toolsManagement.checkForUpdates`, `build.directoryFilters`, `version` to the rest of the system?**
  _15 weakly-connected nodes found - possible documentation gaps or missing edges._