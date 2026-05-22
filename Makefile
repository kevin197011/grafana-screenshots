.PHONY: build up down logs once dry-run restart status trigger health trigger-dry

build:
	docker compose pull

up: build
	docker compose up -d --force-recreate

status:
	@docker compose ps -a
	@echo ""
	@ss -lntp 2>/dev/null | grep 8111 || netstat -lntp 2>/dev/null | grep 8111 || true
	@curl -sf --connect-timeout 2 http://127.0.0.1:8111/health >/dev/null && echo "8111: ok" || echo "8111: 无响应（需容器 Running 且映射 8111，日志应有 HTTP 监听）"

down:
	docker compose down

restart:
	docker compose restart

logs:
	docker compose logs -f

once:
	docker compose run --rm grafana-screenshot once

dry-run:
	docker compose run --rm grafana-screenshot once --dry-run

health: status
	curl -sS http://127.0.0.1:8111/health

trigger: status
	curl -sS --max-time 600 http://127.0.0.1:8111/trigger

trigger-dry: status
	curl -sS --max-time 600 'http://127.0.0.1:8111/trigger?dry-run=1'
