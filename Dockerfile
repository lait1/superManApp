# ──────────────────────────────────────────────────────────────
# superMen — Go backend (multi-stage, CGO disabled, distroless runtime)
# Один статический бинарник: REST API + bot + cron (docs/07-architecture.md §3).
# ──────────────────────────────────────────────────────────────

# ── Stage 1: build ─────────────────────────────────────────
FROM golang:1.26-alpine AS build

WORKDIR /src

# git нужен для go modules, тянущих зависимости по VCS.
RUN apk add --no-cache git

# Кэшируем слой зависимостей: сначала только модули.
COPY go.mod go.sum* ./
RUN go mod download

# Затем исходники.
COPY . .

# Статическая сборка без CGO под Linux.
# Если точка входа переедет в ./cmd/server (docs/07 §3) — поменяй путь сборки тут.
ENV CGO_ENABLED=0 GOOS=linux
ARG BUILD_TARGET=.
RUN go build -trimpath -ldflags="-s -w" -o /out/supermen ${BUILD_TARGET}

# ── Stage 2: runtime ───────────────────────────────────────
# distroless: нет shell/пакетного менеджера, минимальная поверхность атаки.
# Тег :nonroot запускает процесс от непривилегированного пользователя.
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Бинарник и SQL-миграции (на случай встроенного применения миграций).
COPY --from=build /out/supermen /app/supermen
COPY --from=build /src/migrations /app/migrations

ENV PORT=8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/supermen"]
