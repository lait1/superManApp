# 07 — Архитектура

← [06 — Нотификации](./06-notifications.md) · далее → [08 — Модель данных](./08-data-model.md)

Система из трёх частей: **Telegram Mini App** (React-клиент), **Go-бэкенд** (REST API +
бот + cron-шедулер) и **PostgreSQL**. Всё разворачивается за одним доменом + Telegram-бот.

---

## 1. Компонентная схема

```
                ┌──────────────────────────┐
                │        Telegram            │
                │  ┌──────────┐  ┌────────┐ │
   Пользователь │  │ Mini App │  │  Бот    │ │
   ◀───────────▶│  │ (WebView)│  │ @superMen│ │
                │  └────┬─────┘  └───▲────┘ │
                └───────┼────────────┼──────┘
            HTTPS (API) │            │ Bot API (sendMessage)
                        ▼            │
        ┌───────────────────────────┴───────────────┐
        │              Go Backend (1 бинарник)        │
        │                                             │
        │  ┌────────────┐ ┌───────────┐ ┌──────────┐ │
        │  │ REST API    │ │  Game      │ │  Cron     │ │
        │  │ (chi/echo)  │ │  engine    │ │ scheduler │ │
        │  │ + initData  │ │ (XP/gold/  │ │ (daily    │ │
        │  │   auth mw   │ │  quests)   │ │  reports) │ │
        │  └─────┬──────┘ └─────┬─────┘ └─────┬────┘ │
        │        └──────────────┼──────────────┘      │
        │                       ▼                      │
        │              ┌─────────────────┐             │
        │              │  Repository      │             │
        │              │  (pgx / sqlc)    │             │
        │              └────────┬────────┘             │
        └───────────────────────┼──────────────────────┘
                                 ▼
                       ┌──────────────────┐
                       │   PostgreSQL 16   │
                       └──────────────────┘
```

## 2. Клиент — React + TypeScript (Telegram Mini App)

- **Сборка:** Vite. Раздаётся как статика (тем же Go-сервером или отдельным CDN/nginx).
- **Telegram SDK:** `telegram-web-app.js` + `@telegram-apps/sdk` — `initData`, тема, Haptic,
  кнопки. См. [10](./10-telegram-mini-app.md).
- **Состояние:** лёгкий стор (Zustand) + кэш запросов (TanStack Query).
- **Анимации:** Framer Motion + canvas-частицы (см. [05](./05-ui-ux.md)).
- **Сеть:** REST к Go API; в каждый запрос кладётся `initData` для аутентификации.
- **Оффлайн/оптимизм:** чек-ин применяется оптимистично, синкается с ответом сервера.

## 3. Бэкенд — Go

Один бинарник, три обязанности:

### a) REST API
- Роутер `chi` (или `echo`). JSON. Контракт — в [09 — API](./09-api.md).
- **Middleware аутентификации:** валидирует Telegram `initData` (HMAC-SHA256), достаёт
  `telegram_user_id`, находит/создаёт пользователя. Никаких паролей/сессий.
- Dev-fallback: при отсутствии `initData` (запуск в браузере) — заголовок `X-Device-Id`.

### b) Game engine (доменная логика)
- Чистые функции расчёта награды (см. формулы в [03](./03-progression-and-stats.md)):
  `reward(activity, streak, equipment) → {xp, gold, stat, crit, drop}`.
- Прогресс квестов/ачивок, пересчёт уровня/ранга/класса.
- Транзакционность: один чек-ин = одна БД-транзакция (атомарное начисление).

### c) Cron-шедулер
- `robfig/cron` тикает каждые ~5–10 мин, формирует и шлёт daily reports / reminders
  через Telegram Bot API. Логика — в [06 — Нотификации](./06-notifications.md).
- Бот также обрабатывает входящие команды (`/start` → ссылка на Mini App) — webhook или long-poll.

### Структура пакетов (предложение)

```
superMen/
├── main.go                  # точка входа (есть заготовка)
├── cmd/server/              # запуск API + cron + bot
├── internal/
│   ├── api/                 # http-роуты, хендлеры, middleware (initData auth)
│   ├── game/                # доменная логика: xp, gold, quests, stats, classes
│   ├── store/               # репозитории, pgx/sqlc, миграции
│   ├── telegram/            # bot client, initData validation, notifications
│   └── config/              # game config + env
├── migrations/              # SQL-миграции (см. 08-data-model)
└── web/                     # React TMA (Vite)
```

## 4. Поток «чек-ин» (последовательность)

```
Клиент            API (Go)             Game engine        PostgreSQL
  │  POST /checkin   │                      │                 │
  │ (+initData)      │                      │                 │
  ├─────────────────▶│ validate initData    │                 │
  │                  ├─ find/create user     │                 │
  │                  ├─────────────────────▶ │ begin tx        │
  │                  │   compute reward       ├────────────────▶│
  │                  │   update xp/gold/stat  │ insert activity │
  │                  │   advance quests       │ upsert stats    │
  │                  │   roll drop            │ insert tx       │
  │                  │◀───────────────────── │ commit          │
  │  reward event    │                      │                 │
  │◀─────────────────┤                      │                 │
  │ play animation 🎉 │                      │                 │
```

## 5. Конфигурация (env)

| Переменная | Назначение |
|-----------|-----------|
| `DATABASE_URL` | строка подключения PostgreSQL |
| `TELEGRAM_BOT_TOKEN` | токен бота (для initData-валидации и отправки) |
| `TELEGRAM_WEBAPP_URL` | URL Mini App (для web_app-кнопок) |
| `PORT` | порт API (по умолч. 8080) |
| `ENV` | dev/prod (включает device-id fallback в dev) |
| `NOTIFY_TICK` | интервал тика шедулера |

Game-баланс (`XP_BASE`, тиры стрика и т.д. из [03](./03-progression-and-stats.md)) хранится
в БД/конфиге и читается движком — меняется без релиза.

## 6. Деплой (замысел)

```
[ Telegram ] ──HTTPS──▶ [ reverse proxy (nginx/caddy, TLS) ]
                              │            │
                              ▼            ▼
                     [ статика React ]  [ Go API :8080 ]
                                              │
                                              ▼
                                      [ PostgreSQL ]
```

- Telegram Mini App требует **HTTPS** и публичный домен.
- Go-бинарник + Postgres можно поднять через `docker compose`.
- Бот: webhook на тот же домен (`/bot/webhook`) либо long-poll в dev.

## 7. Нефункциональные требования

- **Безопасность:** вся аутентификация — через подпись `initData`; токен бота только на сервере.
- **Идемпотентность:** защита от дабл-чек-инов (кулдаун/дневной потолок, см. [03 §7](./03-progression-and-stats.md#7-анти-абьюз-и-честность)).
- **Наблюдаемость:** структурные логи, метрики (чек-ины, отправки нотификаций).
- **Масштаб:** на старте — один инстанс; stateless API легко масштабируется, cron выносится в отдельный воркер при росте.

---

### Связанные документы
- Схема БД → [08 — Модель данных](./08-data-model.md)
- Контракты API → [09 — API](./09-api.md)
- initData/бот → [10 — Telegram Mini App](./10-telegram-mini-app.md)
