-- ============================================================================
-- superMen — миграция 0001: инициализация схемы (PostgreSQL 16)
-- Источник истины по схеме: docs/08-data-model.md
-- Идемпотентна: все таблицы и индексы создаются с IF NOT EXISTS.
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- 3. Таблицы-справочники (seed) — создаём первыми, на них ссылаются FK
-- ----------------------------------------------------------------------------

-- activities — каталог активностей (из docs/02)
CREATE TABLE IF NOT EXISTS activities (
    key          TEXT PRIMARY KEY,                 -- gym | english | spanish | work | adventure ...
    title        TEXT NOT NULL,
    stat_key     TEXT NOT NULL,                     -- какой стат качает (STR|INT|DIS|VIT|CHA)
    base_xp      INT  NOT NULL,
    base_gold    INT  NOT NULL,
    has_duration BOOLEAN NOT NULL DEFAULT false,
    ref_minutes  INT,                               -- эталон для множителя длительности
    rarity       TEXT NOT NULL DEFAULT 'common',    -- влияет на шанс дропа
    icon         TEXT,
    daily_cap    INT NOT NULL DEFAULT 1             -- сколько раз в день с полной наградой
);

-- quests — каталог квестов
CREATE TABLE IF NOT EXISTS quests (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    type        TEXT NOT NULL,                      -- daily|weekly|chain|side|class|balance
    description TEXT,
    condition   JSONB NOT NULL,                     -- {activity, target, streak_days...}
    reward      JSONB NOT NULL,                     -- {xp, gold, title, item}
    icon        TEXT,
    active      BOOLEAN NOT NULL DEFAULT true
);

-- achievements — каталог ачивок
CREATE TABLE IF NOT EXISTS achievements (
    id          TEXT PRIMARY KEY,
    title       TEXT NOT NULL,
    description TEXT,
    category    TEXT NOT NULL,                      -- start|streak|volume|level|balance|domain|collection
    condition   JSONB NOT NULL,
    reward      JSONB,
    icon        TEXT
);

-- shop_items — каталог товаров магазина (из docs/04)
CREATE TABLE IF NOT EXISTS shop_items (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    slot        TEXT NOT NULL,                      -- weapon|armor|amulet|aura|background|consumable
    rarity      TEXT NOT NULL,                      -- common..legendary
    price       INT,                                -- NULL = не продаётся (только дроп/квест)
    effect      JSONB NOT NULL DEFAULT '{}',        -- {type, stat, value} или {} для косметики
    purchasable BOOLEAN NOT NULL DEFAULT true,
    icon        TEXT
);

-- ----------------------------------------------------------------------------
-- 2. Таблицы — пользовательские данные
-- ----------------------------------------------------------------------------

-- users
CREATE TABLE IF NOT EXISTS users (
    id               BIGSERIAL PRIMARY KEY,
    telegram_user_id BIGINT UNIQUE,                 -- из initData; NULL для dev device-id
    device_id        TEXT UNIQUE,                   -- dev-fallback (браузер)
    username         TEXT,
    timezone         TEXT NOT NULL DEFAULT 'UTC',
    notif_prefs      JSONB NOT NULL DEFAULT '{}',   -- тумблеры из 06-notifications
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at     TIMESTAMPTZ
);

-- characters
CREATE TABLE IF NOT EXISTS characters (
    id                BIGSERIAL PRIMARY KEY,
    user_id           BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name              TEXT NOT NULL DEFAULT 'superMen',
    level             INT  NOT NULL DEFAULT 1,
    xp_total          BIGINT NOT NULL DEFAULT 0,
    gold              BIGINT NOT NULL DEFAULT 0,
    class             TEXT NOT NULL DEFAULT 'adventurer', -- вычисляется движком
    rank              TEXT NOT NULL DEFAULT 'recruit',
    streak_days       INT  NOT NULL DEFAULT 0,
    best_streak       INT  NOT NULL DEFAULT 0,
    last_checkin_date DATE,                          -- для расчёта стрика
    equipped          JSONB NOT NULL DEFAULT '{}',   -- слот → inventory_item_id
    UNIQUE (user_id)
);

-- stats (5 строк на персонажа)
CREATE TABLE IF NOT EXISTS stats (
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    stat_key     TEXT   NOT NULL,                    -- STR | INT | DIS | VIT | CHA
    value        BIGINT NOT NULL DEFAULT 0,
    level        INT    NOT NULL DEFAULT 1,
    PRIMARY KEY (character_id, stat_key)
);

-- activity_logs (история чек-инов)
CREATE TABLE IF NOT EXISTS activity_logs (
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
    local_date    DATE   NOT NULL                    -- дата в TZ пользователя (для стрика/потолка)
);
CREATE INDEX IF NOT EXISTS idx_logs_char_date ON activity_logs(character_id, local_date);

-- quest_progress
CREATE TABLE IF NOT EXISTS quest_progress (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    quest_id     TEXT   NOT NULL REFERENCES quests(id),
    progress     INT    NOT NULL DEFAULT 0,          -- текущее значение к target
    status       TEXT   NOT NULL DEFAULT 'active',   -- active|completed|claimed|expired
    period_key   TEXT,                               -- '2026-06-07' / '2026-W23' для daily/weekly
    completed_at TIMESTAMPTZ,
    UNIQUE (character_id, quest_id, period_key)
);

-- achievement_unlocks
CREATE TABLE IF NOT EXISTS achievement_unlocks (
    character_id   BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    achievement_id TEXT   NOT NULL REFERENCES achievements(id),
    unlocked_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (character_id, achievement_id)
);

-- inventory_items
CREATE TABLE IF NOT EXISTS inventory_items (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    shop_item_id TEXT   NOT NULL REFERENCES shop_items(id),
    acquired_via TEXT   NOT NULL,                    -- purchase|drop|quest|achievement
    quantity     INT    NOT NULL DEFAULT 1,          -- для расходников (заморозка)
    acquired_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_inventory_char ON inventory_items(character_id);

-- transactions (журнал золота)
CREATE TABLE IF NOT EXISTS transactions (
    id           BIGSERIAL PRIMARY KEY,
    character_id BIGINT NOT NULL REFERENCES characters(id) ON DELETE CASCADE,
    amount       INT    NOT NULL,                    -- + начисление / - трата
    reason       TEXT   NOT NULL,                    -- checkin|quest|achievement|purchase|levelup
    ref_id       TEXT,                               -- ссылка на источник
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_transactions_char ON transactions(character_id, created_at);

-- daily_reports (идемпотентность нотификаций)
CREATE TABLE IF NOT EXISTS daily_reports (
    id           BIGSERIAL PRIMARY KEY,
    user_id      BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    report_date  DATE   NOT NULL,
    kind         TEXT   NOT NULL,                    -- daily|streak_reminder|morning|milestone
    sent_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    payload      JSONB,
    UNIQUE (user_id, report_date, kind)              -- не шлём дважды
);

COMMIT;
