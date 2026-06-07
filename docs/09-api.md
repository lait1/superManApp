# 09 — API

← [08 — Модель данных](./08-data-model.md) · далее → [10 — Telegram Mini App](./10-telegram-mini-app.md)

REST API на Go. JSON. Аутентификация — через Telegram `initData` в каждом запросе
(без сессий и паролей). Базовый префикс: `/api/v1`.

---

## 1. Аутентификация

Клиент в каждом запросе передаёт `initData` (строка, выданная Telegram WebApp):

```
Authorization: tma <initData>
```

Сервер валидирует подпись (HMAC-SHA256 по bot token — см. [10](./10-telegram-mini-app.md)),
извлекает `telegram_user_id`, находит/создаёт пользователя и его персонажа.

**Dev-fallback** (запуск в обычном браузере, `ENV=dev`): вместо `initData` —
заголовок `X-Device-Id: <uuid>`; пользователь привязывается к `device_id`.

Ошибки аутентификации → `401`. Формат ошибок единый:

```json
{ "error": { "code": "unauthorized", "message": "invalid initData" } }
```

## 2. Обзор эндпоинтов

| Метод | Путь | Назначение |
|-------|------|-----------|
| `GET`  | `/api/v1/me` | персонаж + статы + золото + стрик (главный экран) |
| `POST` | `/api/v1/checkin` | отметить активность → вернуть reward event |
| `GET`  | `/api/v1/activities` | каталог активностей |
| `GET`  | `/api/v1/quests` | квесты с прогрессом (daily/weekly/chains) |
| `POST` | `/api/v1/quests/{id}/claim` | забрать награду выполненного квеста |
| `GET`  | `/api/v1/achievements` | ачивки (открытые/закрытые) |
| `GET`  | `/api/v1/shop` | товары магазина |
| `POST` | `/api/v1/shop/{itemId}/buy` | купить предмет |
| `GET`  | `/api/v1/inventory` | инвентарь |
| `POST` | `/api/v1/inventory/{id}/equip` | надеть предмет в слот |
| `POST` | `/api/v1/inventory/{id}/unequip` | снять предмет |
| `GET`  | `/api/v1/report/today` | дневная сводка (in-app версия) |
| `PUT`  | `/api/v1/settings/notifications` | тумблеры нотификаций + TZ |

## 3. Ключевые контракты

### GET /me

```json
{
  "character": {
    "name": "superMen", "level": 14, "xpTotal": 6820,
    "xpToNext": 9100, "xpIntoLevel": 6820,
    "gold": 6420, "class": "sage", "rank": "seeker",
    "streakDays": 12, "streakMult": 1.25,
    "equipped": { "weapon": 1201, "amulet": 1188 }
  },
  "stats": [
    { "key": "STR", "value": 540,  "level": 6,  "intoLevel": 120, "toNext": 360 },
    { "key": "INT", "value": 4200, "level": 20, "intoLevel": 50,  "toNext": 980 },
    { "key": "DIS", "value": 1800, "level": 11, "intoLevel": 230, "toNext": 520 },
    { "key": "VIT", "value": 900,  "level": 8,  "intoLevel": 60,  "toNext": 420 },
    { "key": "CHA", "value": 1100, "level": 9,  "intoLevel": 200, "toNext": 460 }
  ],
  "todayCheckins": ["english", "gym"]
}
```

### POST /checkin

Запрос:
```json
{ "activityKey": "english", "durationMin": 45, "note": "урок B2" }
```

Ответ — **reward event** (клиент проигрывает анимацию, см. [05 §6](./05-ui-ux.md#6-сочность-анимации-и-триггеры-награды-)):
```json
{
  "reward": {
    "xp": 113, "gold": 31, "statKey": "INT", "statPoints": 90,
    "isCrit": true, "streakDays": 12, "streakMult": 1.25
  },
  "drop": {
    "itemId": "amulet_owl", "name": "Амулет Совы", "rarity": "rare", "slot": "amulet"
  },
  "levelUp":  { "from": 13, "to": 14 },
  "rankUp":   null,
  "statLevelUp": { "key": "INT", "from": 19, "to": 20 },
  "questsAdvanced": [
    { "id": "quest_polyglot", "progress": 14, "target": 14, "status": "completed" }
  ],
  "achievementsUnlocked": [ "polyglot" ],
  "character": { "level": 14, "xpTotal": 6820, "gold": 6420, "streakDays": 12 }
}
```

> Поля `drop`, `levelUp`, `rankUp`, `statLevelUp` равны `null`/пустым, если события не было.
> Клиент решает, какие анимации проигрывать, по наличию полей.

### GET /quests

```json
{
  "daily":  [ { "id": "daily_english_30", "title": "30 мин английского",
                "progress": 21, "target": 30, "status": "active",
                "reward": { "xp": 60, "gold": 25 } } ],
  "weekly": [ { "id": "weekly_gym_3", "title": "3 тренировки",
                "progress": 2, "target": 3, "status": "active",
                "reward": { "xp": 300, "gold": 150 } } ],
  "chains": [ { "id": "quest_conquistador", "title": "Конкистадор",
                "progress": 18, "target": 30, "status": "active",
                "reward": { "xp": 1000, "gold": 500, "title": "Conquistador" } } ]
}
```

### POST /shop/{itemId}/buy

Успех:
```json
{ "ok": true, "gold": 5220, "inventoryItemId": 1305 }
```
Недостаточно золота → `409`:
```json
{ "error": { "code": "insufficient_gold", "message": "need 1200, have 800" } }
```

### POST /inventory/{id}/equip

```json
{ "ok": true, "equipped": { "weapon": 1201, "amulet": 1305 },
  "statsPreview": { "INT": "+10% XP" } }
```

### PUT /settings/notifications

```json
{ "timezone": "Europe/Madrid",
  "daily": true, "streakReminder": true, "morning": false, "milestone": true,
  "dailyHour": 21 }
```

## 4. Соглашения

- **Версионирование:** префикс `/api/v1`.
- **Идемпотентность чек-ина:** сервер применяет дневной потолок/кулдаун (см.
  [03 §7](./03-progression-and-stats.md#7-анти-абьюз-и-честность)); повторный чек-ин сверх
  потолка вернёт уменьшенную/нулевую награду с флагом `capped: true`.
- **Время:** все timestamp — UTC ISO-8601; «день» считается в TZ пользователя.
- **Ошибки:** единый формат `{ "error": { "code", "message" } }`, коды `400/401/404/409/429/500`.
- **Rate limiting:** мягкий лимит на `/checkin` против абьюза.

---

### Связанные документы
- Откуда `initData` → [10 — Telegram Mini App](./10-telegram-mini-app.md)
- Формулы награды → [03 — Прогрессия и статы](./03-progression-and-stats.md)
- Таблицы под эндпоинты → [08 — Модель данных](./08-data-model.md)
