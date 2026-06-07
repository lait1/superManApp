# 10 — Telegram Mini App

← [09 — API](./09-api.md) · далее → [11 — Роадмап](./11-roadmap.md)

Приложение живёт внутри Telegram как **Mini App** (TMA) — React-клиент в WebView Telegram.
Это даёт три вещи бесплатно: **идентификацию без логина**, **доставку нотификаций** и
**нативную тему/тактильный отклик**.

---

## 1. Почему Telegram Mini App

| Требование | Как решает TMA |
|-----------|----------------|
| «Авторизацию не делаем, сразу главный экран» | Telegram сам передаёт identity (`initData`) — экрана логина нет |
| Ежедневные нотификации | Бот шлёт сообщения в чат (`sendMessage`) — без браузерных разрешений |
| Удобный возврат в петлю | кнопка `web_app` в сообщении открывает Mini App в один тап |
| Нативное ощущение | тема Telegram + Haptic Feedback + main button |

> Граничный момент: Mini App открывается **внутри Telegram**. Для dev/демо в обычном
> браузере предусмотрен fallback по `device_id` (нотификаций там нет).

## 2. Идентификация: `initData`

При запуске Telegram кладёт в WebView строку `initData` (`window.Telegram.WebApp.initData`) —
подписанные данные о пользователе. Клиент шлёт её в каждый запрос (см. [09 §1](./09-api.md#1-аутентификация)),
сервер **обязан проверить подпись** прежде чем доверять `telegram_user_id`.

### Валидация на сервере (Go, схема алгоритма)

```
1. Распарсить initData как query-string в пары ключ=значение.
2. Вынуть поле `hash`, остальные пары отсортировать по ключу.
3. data_check_string = "k1=v1\nk2=v2\n..." (по возрастанию ключей).
4. secret_key = HMAC_SHA256(key="WebAppData", message=BOT_TOKEN)
5. calc_hash = hex( HMAC_SHA256(key=secret_key, message=data_check_string) )
6. Сверить calc_hash == hash  (постоянное по времени сравнение).
7. Проверить auth_date (не слишком старый, напр. ≤ 24 ч) — против replay.
8. Из поля `user` (JSON) взять id, username, language_code.
```

> Это стандартная схема валидации Telegram WebApp initData. Реализуется в
> `internal/telegram` (см. [07 — Архитектура](./07-architecture.md#3-бэкенд--go)).
> Токен бота **только на сервере**, в клиент не попадает.

### Создание пользователя
Первый валидный запрос с новым `telegram_user_id` → создаём `users` + `characters` +
5 строк `stats` (см. [08](./08-data-model.md)). Онбординг — это просто первый заход, без формы регистрации.

## 3. Настройка бота (BotFather)

```
1. /newbot → получить BOT_TOKEN.
2. /setmenubutton или Mini App URL → привязать Mini App к боту.
3. Указать публичный HTTPS-URL Mini App (TELEGRAM_WEBAPP_URL).
4. /start у бота → присылает приветствие + кнопку «Открыть superMen» (web_app).
```

Кнопка, открывающая Mini App из сообщения:
```json
{
  "text": "Открыть superMen",
  "web_app": { "url": "https://app.supermen.example" }
}
```

## 4. SDK и интеграция в React

- Подключить `telegram-web-app.js` (официальный) + опционально `@telegram-apps/sdk`.
- На старте: `WebApp.ready()`, `WebApp.expand()`.
- **Тема:** читать `WebApp.themeParams` → прокинуть в CSS-переменные (фон, текст, акцент),
  чтобы приложение совпадало с темой пользователя (светлая/тёмная).
- **Haptic Feedback:** `WebApp.HapticFeedback.impactOccurred('medium')` на level-up/крит/дроп
  (см. сочность в [05 §6](./05-ui-ux.md#6-сочность-анимации-и-триггеры-награды-)).
- **MainButton/BackButton:** нативные кнопки Telegram для подтверждения чек-ина.
- **Часовой пояс:** определить `Intl.DateTimeFormat().resolvedOptions().timeZone` и сохранить
  через `PUT /settings/notifications` — нужно для слотов рассылки (см. [06](./06-notifications.md)).

## 5. Доставка нотификаций ботом

Ежедневные сообщения шлёт **тот же бот** через Bot API `sendMessage` (см. [06 — Нотификации](./06-notifications.md)).
Поскольку пользователь нажал `/start`, у бота есть право писать ему в чат. Каждое
сообщение содержит инлайн-кнопку `web_app`, открывающую Mini App в нужном месте
(главный экран или чек-ин).

```
Cron (Go) ──▶ Bot API sendMessage(chat_id, text, web_app button) ──▶ чат пользователя
```

`chat_id` для личных сообщений = `telegram_user_id` (хранится в `users`).

## 6. Входящие апдейты бота

| Команда/событие | Реакция |
|-----------------|---------|
| `/start` | приветствие + кнопка «Открыть superMen» |
| `/help` | краткая справка |
| нажатие web_app кнопки | открытие Mini App (обрабатывает Telegram) |

Приём апдейтов: **webhook** (`/bot/webhook` на том же домене) в prod, long-poll в dev.

## 7. Dev-fallback (без Telegram)

Чтобы разрабатывать UI в обычном браузере:
- если `window.Telegram.WebApp.initData` пустой → клиент генерит/хранит `device_id` (localStorage)
  и шлёт его в `X-Device-Id`;
- сервер при `ENV=dev` принимает device-id и привязывает пользователя к нему;
- нотификации в этом режиме недоступны (нет chat_id).

## 8. Чек-лист запуска TMA

- [ ] Бот создан, `BOT_TOKEN` в env сервера.
- [ ] Mini App URL (HTTPS) привязан в BotFather.
- [ ] Сервер валидирует `initData` (подпись + `auth_date`).
- [ ] Клиент шлёт `Authorization: tma <initData>`.
- [ ] Тема и Haptic подключены.
- [ ] TZ пользователя сохраняется.
- [ ] Webhook бота настроен (prod).

---

### Связанные документы
- Аутентификация запросов → [09 — API §1](./09-api.md#1-аутентификация)
- Где валидируется initData → [07 — Архитектура](./07-architecture.md)
- Контент и расписание нотификаций → [06 — Нотификации](./06-notifications.md)
