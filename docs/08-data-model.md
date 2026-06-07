# 08 — Модель данных (PostgreSQL)

← [07 — Архитектура](./07-architecture.md) · далее → [09 — API](./09-api.md)

Схема для PostgreSQL 16. Названия — на английском (для совместимости с кодом), комментарии —
по делу. DDL ниже — наброски для первой миграции, не финал.

---

## 1. ER-диаграмма (обзор)

```
                    ┌──────────┐
                    │  users    │ (telegram_user_id, tz, notif_prefs)
                    └────┬─────┘
                         │ 1
              ┌──────────┼───────────────────────────┐
              │ 1        │ 1                          │ 1
        ┌─────▼────┐ ┌───▼──────┐               ┌─────▼─────────┐
        │characters│ │  stats    │ (5 строк/юзер) │ daily_reports │
        └────┬─────┘ └──────────┘               └───────────────┘
             │ 1
   ┌─────────┼──────────────┬──────────────┬─────────────────┐
   │ N       │ N            │ N            │ N               │ N
┌──▼──────┐┌─▼──────────┐┌──▼──────────┐┌──▼────────────┐┌───▼─────────┐
│activity ││quest_       ││achievement_ ││inventory_     ││transactions │
│_logs    ││progress     ││unlocks      ││items          ││(gold ledger)│
└─────────┘└────┬───────┘└──────┬──────┘└──────┬────────┘└─────────────┘
                │ ref            │ ref          │ ref
          ┌─────▼────┐    ┌──────▼──────┐ ┌─────▼──────┐
          │ quests    │    │achievements │ │ shop_items │   ← справочники (seed)
          │(catalog)  │    │ (catalog)   │ │ (catalog)  │
          └───────────┘    └─────────────┘ └────────────┘
          ┌──────────────┐
          │ activities    │  ← справочник активностей (catalog)
          │ (catalog)     │
          └──────────────┘
```

Разделение на **пользовательские данные** и **справочники (каталоги)**: активности,
квесты, ачивки, товары магазина — это seed-данные, общие для всех.

## 2. Таблицы — пользовательские данные

### users
```sql
CREATE TABLE users (
    id               BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT UNIQUE,            -- из initData; NULL для dev device-id
    device_id        TEXT UNIQUE,              -- dev-fallback (браузер)
    username         TEXT,
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    notif_prefs      JSONB NOT NULL DEFAULT '{}',  -- тумблеры из 06-notifications
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at     TIMESTAMPTZ
);
```

### characters
```sql
CREATE TABLE characters (
    id              BIGSERIAL PRIMARY KEY,
    user_id         BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            TEXT NOT NULL DEFAULT 'superMen',
    level           INT  NOT NULL DEFAULT 1,
    xp_total        BIGINT NOT NULL DEFAULT 0,
    gold            BIGINT NOT NULL DEFAULT 0,
    class           TEXT NOT NULL DEFAULT 'adventurer', -- вычисляется движком
    rank            TEXT NOT NULL DEFAULT 'recruit',
    streak_days     INT  NOT NULL DEFAULT 0,
    best_streak     INT  NOT NULL DEFAULT 0,
    last_checkin_date DATE,                     -- для расчёта стрика
    equipped        JSONB NOT NULL DEFAULT '{}', -- слот → inventory_item_id
    UNIQUE (user_id)
);
```

### stats (5 строк на персонажа)
```sql
CREATE TABLE stats (
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    stat_key     TEXT   NOT NULL,    -- STR | INT | DIS | VIT | CHA
    value        BIGINT NOT NULL DEFAULT 0,
    level        INT    NOT NULL DEFAULT 1,
    PRIMARY KEY (character_id, stat_key)
);
```

### activity_logs (история чек-инов)
```sql
CREATE TABLE activity_logs (
    id            BIGSERIAL PRIMARY KEY,
    character_id  BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    activity_key  TEXT   NOT NULL REFERENCES activities(key),
    duration_min  INT,
    note          TEXT,
    xp_awarded    INT    NOT NULL,
    gold_awarded  INT    NOT NULL,
    stat_awarded  INT    NOT NULL,
    is_crit       BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    local_date    DATE   NOT NULL          -- дата в TZ пользователя (для стрика/потолка)
);
CREATE INDEX idx_logs_char_date ON activity_logs(character_id, local_date);
```

### quest_progress
```sql
CREATE TABLE quest_progress (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    quest_id     TEXT   NOT NULL REFERENCES quests(id),
    progress     INT    NOT NULL DEFAULT 0,    -- текущее значение к target
    status       TEXT   NOT NULL DEFAULT 'active', -- active|completed|claimed|expired
    period_key   TEXT,                          -- '2026-06-07' / '2026-W23' для daily/weekly
    completed_at TIMESTAMPTZ,
    UNIQUE (character_id, quest_id, period_key)
);
```

### achievement_unlocks
```sql
CREATE TABLE achievement_unlocks (
    character_id   BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    achievement_id TEXT   NOT NULL REFERENCES achievements(id),
    unlocked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, achievement_id)
);
```

### inventory_items
```sql
CREATE TABLE inventory_items (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    shop_item_id TEXT   NOT NULL REFERENCES shop_items(id),
    acquired_via TEXT   NOT NULL,        -- purchase|drop|quest|achievement
    quantity     INT    NOT NULL DEFAULT 1,  -- для расходников (заморозка)
    acquired_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### transactions (журнал золота)
```sql
CREATE TABLE transactions (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    amount       INT    NOT NULL,        -- + начисление / - трата
    reason       TEXT   NOT NULL,        -- checkin|quest|achievement|purchase|levelup
    ref_id       TEXT,                   -- ссылка на источник
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### daily_reports (идемпотентность нотификаций)
```sql
CREATE TABLE daily_reports (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_date  DATE   NOT NULL,
    kind         TEXT   NOT NULL,        -- daily|streak_reminder|morning|milestone
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload      JSONB,
    UNIQUE (user_id, report_date, kind)  -- не шлём дважды
);
```

## 3. Таблицы-справочники (seed)

### activities (каталог из [02](./02-activities-and-quests.md))
```sql
CREATE TABLE activities (
    key         TEXT PRIMARY KEY,        -- gym | english | spanish | work | adventure ...
    title       TEXT NOT NULL,
    stat_key    TEXT NOT NULL,           -- какой стат качает
    base_xp     INT  NOT NULL,
    base_gold   INT  NOT NULL,
    has_duration BOOLEAN NOT NULL DEFAULT false,
    ref_minutes INT,                     -- эталон для множителя длительности
    rarity      TEXT NOT NULL DEFAULT 'common', -- влияет на шанс дропа
    icon        TEXT,
    daily_cap   INT NOT NULL DEFAULT 1   -- сколько раз в день с полной наградой
);
```

### quests
```sql
CREATE TABLE quests (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    type        TEXT NOT NULL,           -- daily|weekly|chain|side|class|balance
    description TEXT,
    condition   JSONB NOT NULL,          -- {activity, target, streak_days...}
    reward      JSONB NOT NULL,          -- {xp, gold, title, item}
    icon        TEXT,
    active      BOOLEAN NOT NULL DEFAULT true
);
```

### achievements
```sql
CREATE TABLE achievements (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    category    TEXT NOT NULL,           -- start|streak|volume|level|balance|domain|collection
    condition   JSONB NOT NULL,
    reward      JSONB,
    icon        TEXT
);
```

### shop_items
```sql
CREATE TABLE shop_items (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slot        TEXT NOT NULL,           -- weapon|armor|amulet|aura|background|consumable
    rarity      TEXT NOT NULL,           -- common..legendary
    price       INT,                     -- NULL = не продаётся (только дроп/квест)
    effect      JSONB NOT NULL DEFAULT '{}', -- {type, stat, value} или {} для косметики
    purchasable BOOLEAN NOT NULL DEFAULT true,
    icon        TEXT
);
```

## 4. Замечания по схеме

- **Денормализация для скорости чтения:** `characters.level/gold/streak` держим прямо в строке
  персонажа (главный экран читается одним запросом), а `transactions`/`activity_logs` — это
  аудит-журнал источника истины.
- **JSONB для гибких полей** (`condition`, `reward`, `effect`, `equipped`, `notif_prefs`) —
  чтобы менять правила без миграций схемы. Тяжёлую логику валидируем в Go.
- **`local_date`** считается в таймзоне пользователя — критично для стриков и дневных потолков.
- **Идемпотентность нотификаций** — уникальный индекс в `daily_reports`.
- **Каталоги версионируются миграциями/сидами** — баланс правится централизованно.

## 5. Стартовые сиды

Первая миграция наполняет каталоги: активности (из [02](./02-activities-and-quests.md)),
базовый набор квестов и ачивок, товары магазина (из [04](./04-economy-and-shop.md)),
а также 5 строк `stats` создаются при создании персонажа.

---

### Связанные документы
- Откуда значения каталогов → [02](./02-activities-and-quests.md), [04](./04-economy-and-shop.md)
- Кто пишет в таблицы → [09 — API](./09-api.md)
- Как создаётся пользователь → [10 — Telegram Mini App](./10-telegram-mini-app.md)
