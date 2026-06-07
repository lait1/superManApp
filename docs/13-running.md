# 13 — Запуск и разработка

← [12 — Дизайн персонажа](./12-character-design.md) · [README](../README.md) · связано: [07 — Архитектура](./07-architecture.md), [10 — Telegram Mini App](./10-telegram-mini-app.md)

Пошаговый гайд: как поднять superMen локально (офлайн на memory-store, либо с PostgreSQL),
как сгенерировать ассеты и как привязать Telegram-бота. Все команды собраны в `Makefile`;
переменные окружения — в `.env` (шаблон `.env.example`, описание — в [07 §5](./07-architecture.md#5-конфигурация-env)).

---

## 0. Предварительные требования

| Инструмент | Зачем | Минимум |
|-----------|-------|---------|
| Go | бэкенд (один бинарник: API + bot + cron, [07 §3](./07-architecture.md#3-бэкенд--go)) | 1.26 |
| Node.js + npm | сборка React Mini App (`web/`, [10](./10-telegram-mini-app.md)) | Node 20+ |
| Docker + Docker Compose | поднять стек целиком (Postgres + backend + web) | любой свежий |
| `psql` | применение миграций через `make migrate` | из пакета PostgreSQL client |

Первым делом скопируй шаблон окружения:

```bash
cp .env.example .env
```

Список целей `make`:

```bash
make help
```

---

## 1. Офлайн-запуск (memory-store, без БД)

Самый быстрый способ покрутить бэкенд: `ENV=dev` + пустой `DATABASE_URL` → данные
живут в памяти процесса, БД не нужна. Аутентификация работает через device-id fallback
(см. [10 §7](./10-telegram-mini-app.md#7-dev-fallback-без-telegram)) — Mini App открывается
в обычном браузере, без Telegram. Нотификации в этом режиме недоступны (нет `chat_id`).

```bash
# 1) Бэкенд на :8080 (memory-store)
make dev

# 2) В соседнем терминале — фронт (Vite dev-сервер)
make web-dev
```

- API: `http://localhost:8080` (контракты — [09 — API](./09-api.md)).
- Mini App: URL, который напечатает Vite (обычно `http://localhost:5173`).
- Клиент при пустом `initData` шлёт `X-Device-Id` — сервер при `ENV=dev` принимает его
  и привязывает пользователя ([10 §7](./10-telegram-mini-app.md#7-dev-fallback-без-telegram)).

> Это то, что нужно для итерации по UI и core loop ([11 — Фаза 1](./11-roadmap.md#фаза-1--mvp-замкнуть-петлю))
> без поднятия инфраструктуры.

---

## 2. Запуск с PostgreSQL + миграциями

Когда нужны персистентность, стрики и нотификации — поднимаем PostgreSQL и применяем
схему из [08 — Модель данных](./08-data-model.md).

### Вариант A — БД в Docker, бэкенд/фронт локально

```bash
# 1) Только Postgres из compose-стека
docker compose up -d postgres

# 2) Прописать DATABASE_URL в .env на localhost, например:
#    DATABASE_URL=postgres://supermen:supermen@localhost:5432/supermen?sslmode=disable

# 3) Применить миграции (migrations/*.sql по порядку через psql)
make migrate

# 4) Запустить бэкенд (теперь он увидит непустой DATABASE_URL и пойдёт в БД)
make dev
```

> `make migrate` берёт `DATABASE_URL` из `.env`, прогоняет файлы `migrations/*.sql`
> по возрастанию имени с `ON_ERROR_STOP=1`. Первая миграция создаёт таблицы и наполняет
> каталоги-сиды (активности, квесты, ачивки, товары — [08 §5](./08-data-model.md#5-стартовые-сиды)).
>
> Примечание: при **первой** инициализации тома Postgres compose сам прогоняет
> `migrations/` через `docker-entrypoint-initdb.d`. На уже существующем томе автозапуска
> не будет — применяй `make migrate` вручную.

### Вариант B — весь стек в Docker

Поднимает Postgres + backend (из `Dockerfile`, multi-stage, distroless, без CGO) + web:

```bash
make up        # = docker compose up -d --build
make logs      # хвост логов
make ps        # статус контейнеров
make down      # остановить (тома и данные сохраняются)
make clean     # снести стек вместе с томами (УДАЛИТ данные БД)
```

Внутри сети compose host базы = имя сервиса `postgres` (не `localhost`) — для контейнеров
это уже зашито в дефолтном `DATABASE_URL` в `docker-compose.yml`.

После `make up`:
- API — `http://localhost:8080`,
- Mini App (Vite preview) — `http://localhost:5173`.

---

## 3. Генерация ассетов

Пиксель-арт спрайты персонажа и предметов производятся через `cmd/genassets`
(арт-пайплайн и формат — [12 — Дизайн персонажа](./12-character-design.md)):

```bash
make genassets        # = go run ./cmd/genassets
```

Результат складывается в `web/public/assets/character/*.png`. По умолчанию эти PNG
**игнорируются гитом** (`.gitignore`) — считаем их генерируемыми. Если решишь хранить
готовые ассеты в репозитории, убери соответствующую строку из `.gitignore`.

Рендер слоёв (paper-doll, z-порядок, якоря) описан в [12 §5](./12-character-design.md#5-система-слоёв-paper-doll-на-пиксельной-сетке)
и [12 §12](./12-character-design.md#12-рендер-во-фронтенде-react).

---

## 4. Запуск нового приложения в Telegram — пошагово

Полная теория — в [10 — Telegram Mini App](./10-telegram-mini-app.md). Здесь — рабочий
пошаговый гайд «с нуля до открытия Mini App в Telegram».

> **Как связаны части.** Фронт (Vite, `:5173`) обращается к API по относительному
> `/api/v1`, а Vite-dev **уже проксирует** `/api → localhost:8080` (см. `web/vite.config.ts`).
> Поэтому в Telegram достаточно прокинуть HTTPS-туннель к **одному** порту `:5173` — он
> сам проксирует API на бэкенд. (Бэкенд статику не раздаёт — это путь для прода, см. §4.6.)

### Шаг 1. Создать бота и получить токен

1. В Telegram открой [@BotFather](https://t.me/BotFather) → `/newbot` → задай имя и username
   → получи **BOT_TOKEN** (вида `1234567890:AA...`).
2. Положи токен в `.env`:
   ```env
   TELEGRAM_BOT_TOKEN=1234567890:AAyourtoken
   ```
   Токен живёт только на сервере (нужен для валидации `initData` и отправки сообщений),
   в клиент не попадает ([10 §2](./10-telegram-mini-app.md#2-идентификация-initdata)).

### Шаг 2. Запустить бэкенд (с токеном)

```bash
make dev
#   = ENV=dev PORT=8080 ... go run .   (memory-store, без БД)
```
- `ENV=dev` оставляем — он разрешает и `initData` (из Telegram), и device-id fallback
  (для проверки в браузере). Бэкенд подхватит `TELEGRAM_BOT_TOKEN` из `.env`.
- Хочешь сохранять прогресс между перезапусками — подними Postgres по §2 и задай
  `DATABASE_URL` (тогда нужен и pgx-драйвер, см. §4.6).

### Шаг 3. Запустить фронт (Vite)

```bash
make web-dev
#   = cd web && npm install && npm run dev   → http://localhost:5173
```

### Шаг 4. Поднять HTTPS-туннель к :5173

Telegram пускает в Mini App **только публичный HTTPS-URL** — локальный туннель даёт его:

```bash
# вариант cloudflared (без аккаунта, быстрый):
cloudflared tunnel --url http://localhost:5173

# или ngrok:
ngrok http 5173
```
Скопируй выданный `https://<что-то>.trycloudflare.com` (или `*.ngrok-free.app`).

> ℹ️ **Туннель и allowedHosts.** Домены `*.trycloudflare.com`, `*.ngrok-free.app`,
> `*.ngrok.io` уже разрешены в `web/vite.config.ts` (`allowedHosts`). Если используешь
> другой туннель-домен — добавь его туда и перезапусти `make web-dev`.

### Шаг 5. Привязать Mini App к боту (BotFather)

Положи тот же HTTPS-URL в `.env` (его использует бот для `web_app`-кнопок в нотификациях):
```env
TELEGRAM_WEBAPP_URL=https://<твой-туннель>
```
Затем в [@BotFather](https://t.me/BotFather) — любой из способов:
- **Кнопка-меню в чате:** `/setmenubutton` → выбери бота → текст кнопки + укажи HTTPS-URL.
- **Отдельное Mini App (даёт `t.me`-ссылку):** `/newapp` → выбери бота → заполни → укажи Web App URL.

(Бэкенд перезапусти, чтобы он увидел новый `TELEGRAM_WEBAPP_URL`.)

### Шаг 6. Открыть и проверить

1. Открой своего бота в Telegram → `/start` → придёт приветствие с кнопкой
   «Открыть superMen» ([10 §6](./10-telegram-mini-app.md#6-входящие-апдейты-бота)).
2. Нажми кнопку-меню / кнопку из сообщения → Mini App открывается **внутри Telegram**.
3. Сервер валидирует `initData` (HMAC-SHA256 + `auth_date`) и заводит/находит пользователя —
   ты сразу на главном экране (без логина).
4. Сделай чек-ин → проверь начисление XP/золота/анимацию; назавтра придёт daily report
   (для нотификаций бэкенд с токеном должен быть запущен; пользователь должен был нажать `/start`).

Чек-лист запуска TMA целиком — [10 §8](./10-telegram-mini-app.md#8-чек-лист-запуска-tma).

### Промежуточная проверка без Telegram (опционально)

Перед туннелем убедись, что фронт жив локально: открой `http://localhost:5173` в браузере —
в dev-режиме клиент пойдёт по `X-Device-Id`, и петля чек-ина должна работать.

### 4.6 Путь в прод (когда базовый функционал подтверждён)

Для постоянного хостинга (не туннель):
- собрать статику фронта `make web-build` (`web/dist`) и раздавать её + `/api` с **одного
  origin** через reverse-proxy (nginx/caddy, TLS) — тогда относительный `/api/v1` работает
  без прокси Vite ([07 §6](./07-architecture.md#6-деплой-замысел));
- *(follow-up)* добавить раздачу `web/dist` самим бэкендом (`embed` + `http.FileServer`),
  чтобы был один бинарник без отдельного веб-сервера;
- подключить **pgx-драйвер** для Postgres (blank-import `github.com/jackc/pgx/v5/stdlib`
  в `main.go`), иначе `sql.Open("pgx", …)` не резолвится — см. комментарии в
  `internal/store/postgres/postgres.go` и `main.go`.

---

## 5. Переменные окружения

Все переменные и их назначение — в [07 §5](./07-architecture.md#5-конфигурация-env).
Шаблон — `.env.example`. Сводка:

| Переменная | Назначение | Заметка по запуску |
|-----------|-----------|---------------------|
| `DATABASE_URL` | подключение к PostgreSQL | пусто + `ENV=dev` → memory-store (§1) |
| `TELEGRAM_BOT_TOKEN` | токен бота (initData + отправка) | из BotFather (§4) |
| `TELEGRAM_WEBAPP_URL` | URL Mini App для `web_app`-кнопок | публичный HTTPS (§4) |
| `PORT` | порт API | по умолчанию `8080` |
| `ENV` | `dev` / `prod` | `dev` включает device-id fallback (§1) |
| `NOTIFY_TICK` | интервал тика cron-шедулера | напр. `5m` ([06](./06-notifications.md)) |

---

## 6. Сборка и тесты

```bash
make build    # статический бинарник ./bin/supermen (CGO off)
make test     # go test ./...
make web-build # прод-статика web/dist
```

---

## 7. Шпаргалка по сценариям

| Хочу… | Команды |
|-------|---------|
| Покрутить UI без БД | `make dev` + `make web-dev` (§1) |
| Полноценно с БД локально | `docker compose up -d postgres` → `make migrate` → `make dev` (§2A) |
| Поднять всё в Docker | `make up` (§2B) |
| Пересобрать ассеты | `make genassets` (§3) |
| Подключить Telegram | заполнить `.env` + BotFather (§4) |

---

### Связанные документы
- Конфигурация и компоненты → [07 — Архитектура](./07-architecture.md)
- initData, бот, dev-fallback → [10 — Telegram Mini App](./10-telegram-mini-app.md)
- Схема БД и сиды → [08 — Модель данных](./08-data-model.md)
- Что и в каком порядке делать → [11 — Роадмап](./11-roadmap.md)
- Формат ассетов персонажа → [12 — Дизайн персонажа](./12-character-design.md)
