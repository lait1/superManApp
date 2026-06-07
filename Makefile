# ──────────────────────────────────────────────────────────────
# superMen — Makefile. Цели для бэкенда (Go), фронта (web/) и Docker.
# Подробный гайд: docs/13-running.md.
# ──────────────────────────────────────────────────────────────

# Подхватываем .env, если он есть (для migrate и пр.), не падая без него.
-include .env
export

# Дефолты на случай, если переменная не задана в .env.
ENV         ?= dev
PORT        ?= 8080
NOTIFY_TICK ?= 5m
DATABASE_URL ?=

.DEFAULT_GOAL := help

.PHONY: help dev build test genassets migrate web-dev web-build up down logs ps clean

## help: показать список целей
help:
	@echo "superMen — доступные цели make:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

# ── Backend (Go) ───────────────────────────────────────────

## dev: запустить бэкенд локально в dev (ENV=dev, memory-store без БД)
dev:
	ENV=dev PORT=$(PORT) NOTIFY_TICK=$(NOTIFY_TICK) go run .

## build: собрать статический бинарник в ./bin/supermen (без CGO)
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o bin/supermen .

## test: прогнать все Go-тесты
test:
	go test ./...

## genassets: сгенерировать пиксель-арт ассеты (см. docs/12-character-design.md)
genassets:
	go run ./cmd/genassets

## migrate: применить migrations/*.sql к $DATABASE_URL через psql
migrate:
	@if [ -z "$(DATABASE_URL)" ]; then \
		echo "ОШИБКА: DATABASE_URL не задан. Заполни .env (cp .env.example .env)."; \
		exit 1; \
	fi
	@echo "Применяю миграции к $(DATABASE_URL)"
	@for f in $$(ls migrations/*.sql | sort); do \
		echo ">> $$f"; \
		psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$$f" || exit 1; \
	done
	@echo "Миграции применены."

# ── Frontend (web/) ────────────────────────────────────────

## web-dev: запустить Vite dev-сервер фронта (web/)
web-dev:
	cd web && npm install && npm run dev

## web-build: собрать прод-статику фронта (web/dist)
web-build:
	cd web && npm install && npm run build

# ── Docker ─────────────────────────────────────────────────

## up: поднять весь стек (postgres + backend + web) через docker compose
up:
	docker compose up -d --build

## down: остановить стек (тома сохраняются)
down:
	docker compose down

## logs: хвост логов всех сервисов
logs:
	docker compose logs -f

## ps: статус контейнеров
ps:
	docker compose ps

## clean: убрать стек вместе с томами (УДАЛИТ данные БД) и локальные артефакты
clean:
	docker compose down -v
	rm -rf bin dist
