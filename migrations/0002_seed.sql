-- ============================================================================
-- superMen — миграция 0002: сиды каталогов (PostgreSQL 16)
-- Источники значений:
--   activities  → docs/02-activities-and-quests.md §1
--   quests      → docs/02-activities-and-quests.md §2
--   achievements→ docs/02-activities-and-quests.md §3
--   shop_items  → docs/04-economy-and-shop.md §5–7
-- Идемпотентна: все INSERT'ы используют ON CONFLICT DO NOTHING.
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- activities (каталог из docs/02 §1)
--   stat_key: STR | INT | DIS | VIT | CHA
--   base_xp / base_gold — точно из таблицы каталога
--   rarity: common по умолчанию; редкие/эпические — adventure / speech
--   has_duration: gym (можно указать длительность), а также трекаемые по времени
-- ----------------------------------------------------------------------------
INSERT INTO activities (key, title, stat_key, base_xp, base_gold, has_duration, ref_minutes, rarity, icon, daily_cap) VALUES
    ('gym',        'Спортзал',                'STR', 50, 30, true,  60, 'common',   '🏋️', 1),
    ('cardio',     'Кардио / бег',            'STR', 40, 25, true,  30, 'common',   '🏃', 1),
    ('english',    'Английский',              'INT', 40, 20, true,  30, 'common',   '🧠', 1),
    ('spanish',    'Испанский',               'INT', 40, 20, true,  30, 'common',   '🗣️', 1),
    ('reading',    'Чтение',                  'INT', 25, 15, true,  30, 'common',   '📖', 1),
    ('work',       'Рабочая фокус-сессия',    'DIS', 35, 20, true,  90, 'common',   '💼', 2),
    ('wakeup',     'Ранний подъём',           'DIS', 20, 10, false, NULL,'common',  '⏰', 1),
    ('meditation', 'Медитация',               'VIT', 25, 15, true,  20, 'common',   '🧘', 1),
    ('sleep',      'Здоровый сон',            'VIT', 20, 10, true,  450,'common',   '😴', 1),
    ('nutrition',  'Здоровое питание',        'VIT', 20, 10, false, NULL,'common',  '🥗', 1),
    ('adventure',  'Приключение / новый опыт','CHA', 70, 50, false, NULL,'rare',    '🧭', 1),
    ('meetup',     'Встреча / нетворкинг',    'CHA', 35, 20, false, NULL,'common',  '🤝', 1),
    ('speech',     'Публичное выступление',   'CHA', 80, 60, false, NULL,'epic',    '🎤', 1)
ON CONFLICT (key) DO NOTHING;

-- ----------------------------------------------------------------------------
-- quests — КАНОН: internal/store/memory/seed.go (seedQuests, 9 квестов)
--   type: daily | weekly | chain | side | class | balance
--   condition JSONB: ключ "activity" (строка, вкл. мета-группу "language");
--                    метрики "minutes" / "target" / "count" / "streak_days"
--   reward JSONB: {xp, gold, title, item} (domain.QuestReward, omitempty)
-- ----------------------------------------------------------------------------
INSERT INTO quests (id, title, type, description, condition, reward, icon, active) VALUES
    -- Daily
    ('daily_english_30', '30 мин английского',
        'daily', 'Сделай 30 минут английского сегодня',
        '{"activity": "english", "minutes": 30}'::jsonb,
        '{"xp": 60, "gold": 25}'::jsonb,
        '🧠', true),
    ('daily_work_2', '2 фокус-сессии',
        'daily', 'Проведи 2 рабочие фокус-сессии',
        '{"activity": "work", "target": 2}'::jsonb,
        '{"xp": 100, "gold": 50}'::jsonb,
        '💼', true),

    -- Weekly
    ('weekly_gym_3', '3 тренировки',
        'weekly', 'Сходи в спортзал 3 раза за неделю',
        '{"activity": "gym", "target": 3}'::jsonb,
        '{"xp": 300, "gold": 150}'::jsonb,
        '🏋️', true),
    ('weekly_lang_5', '5 языковых занятий',
        'weekly', '5 занятий любым языком за неделю',
        '{"activity": "language", "target": 5}'::jsonb,
        '{"xp": 300, "gold": 150}'::jsonb,
        '🗣️', true),

    -- Chain
    ('chain_polyglot', 'Полиглот',
        'chain', '14 дней любого языка подряд',
        '{"activity": "language", "streak_days": 14}'::jsonb,
        '{"xp": 700, "gold": 300, "title": "Полиглот"}'::jsonb,
        '🧠', true),
    ('chain_conquistador', 'Конкистадор',
        'chain', 'Занимайся испанским 30 дней подряд',
        '{"activity": "spanish", "streak_days": 30}'::jsonb,
        '{"xp": 1000, "gold": 500, "title": "Conquistador", "item": "legendary_cape_conquistador"}'::jsonb,
        '🗣️', true),
    ('chain_iron', 'Железный',
        'chain', '50 тренировок суммарно',
        '{"activity": "gym", "count": 50}'::jsonb,
        '{"xp": 1000, "gold": 500, "item": "armor_titan"}'::jsonb,
        '🏋️', true),
    ('chain_discipline', 'Несокрушимый',
        'chain', '21 день подряд с фокус-сессией',
        '{"activity": "work", "streak_days": 21}'::jsonb,
        '{"xp": 800, "gold": 400, "title": "Несокрушимый"}'::jsonb,
        '💼', true),

    -- Side / Приключение
    ('side_discovery', 'Открытие',
        'side', 'Попробуй что-то, чего никогда не делал',
        '{"activity": "adventure", "target": 1}'::jsonb,
        '{"xp": 150, "gold": 100}'::jsonb,
        '🧭', true)
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- achievements — КАНОН: internal/store/memory/seed.go (seedAchievements, 13 шт)
--   category: start | streak | volume | level | balance | domain | collection
--   id'ы совпадают с хардкодом движка (internal/game/engine.go):
--     first_checkin, first_levelup, first_drop, streak_7/30/100,
--     volume_gym_100, level_10/25/50/100, harmony, polyglot
--   reward в seed.go == nil у first_drop и volume_gym_100 → колонка reward
--   nullable (0001), пишем NULL там, где награды нет.
-- ----------------------------------------------------------------------------
INSERT INTO achievements (id, title, description, category, condition, reward, icon) VALUES
    -- Старт
    ('first_checkin', 'Первый шаг', 'Первый чек-ин',
        'start', '{"checkins": 1}'::jsonb, '{"xp": 50}'::jsonb, '👣'),
    ('first_levelup', 'Рост', 'Первый level-up',
        'start', '{"level": 2}'::jsonb, '{"gold": 50}'::jsonb, '⬆️'),
    ('first_drop', 'Находка', 'Первый дроп шмота',
        'start', '{"drops": 1}'::jsonb, NULL, '🎁'),

    -- Стрики
    ('streak_7', 'Неделя огня', 'Стрик 7 дней',
        'streak', '{"streak_days": 7}'::jsonb, '{"xp": 100, "gold": 50}'::jsonb, '🔥'),
    ('streak_30', 'Месяц силы', 'Стрик 30 дней',
        'streak', '{"streak_days": 30}'::jsonb, '{"xp": 500, "gold": 250}'::jsonb, '🔥'),
    ('streak_100', 'Сотня', 'Стрик 100 дней',
        'streak', '{"streak_days": 100}'::jsonb, '{"xp": 2000, "gold": 1000}'::jsonb, '🔥'),

    -- Объём
    ('volume_gym_100', '100 тренировок', '100 тренировок суммарно',
        'volume', '{"activity": "gym", "count": 100}'::jsonb, NULL, '🏋️'),

    -- Уровни
    ('level_10', 'LVL 10', 'Достигни 10 уровня',
        'level', '{"level": 10}'::jsonb, '{"gold": 200}'::jsonb, '🎖️'),
    ('level_25', 'LVL 25', 'Достигни 25 уровня',
        'level', '{"level": 25}'::jsonb, '{"gold": 500}'::jsonb, '🎖️'),
    ('level_50', 'LVL 50', 'Достигни 50 уровня',
        'level', '{"level": 50}'::jsonb, '{"gold": 1000}'::jsonb, '🏅'),
    ('level_100', 'LVL 100', 'Достигни 100 уровня',
        'level', '{"level": 100}'::jsonb, '{"gold": 2500}'::jsonb, '👑'),

    -- Баланс
    ('harmony', 'Гармония', 'Все статы ≥ LVL 10',
        'balance', '{"all_stats_level": 10}'::jsonb, '{"title": "Гармония"}'::jsonb, '☯️'),

    -- Домен
    ('polyglot', 'Полиглот', 'Заверши цепочку «Полиглот»',
        'domain', '{"quest": "chain_polyglot"}'::jsonb, '{"title": "Полиглот"}'::jsonb, '🧠')
ON CONFLICT (id) DO NOTHING;

-- ----------------------------------------------------------------------------
-- shop_items — КАНОН: internal/store/memory/seed.go (seedShopItems; здесь
-- исходные 7 позиций, остальной каталог добавляет 0004_shop_items.sql)
--   slot: weapon|armor|amulet|aura|background|consumable
--   rarity: common..legendary
--   price NULL + purchasable=false → только дроп/квест (легендарки)
--   effect JSONB — domain.ItemEffect {type, stat, value, charges} (omitempty):
--     xp_mult → {type, stat, value}; streak_shield → {type, charges};
--     cosmetic → {type} (без stat/value)
-- ----------------------------------------------------------------------------
INSERT INTO shop_items (id, name, slot, rarity, price, effect, purchasable, icon) VALUES
    ('amulet_owl', 'Амулет Совы', 'amulet', 'rare', 1200,
        '{"type": "xp_mult", "stat": "INT", "value": 0.10}'::jsonb, true, '📿'),
    ('blade_focus', 'Клинок Фокуса', 'weapon', 'epic', 3500,
        '{"type": "xp_mult", "stat": "DIS", "value": 0.10}'::jsonb, true, '⚔️'),
    ('bg_neon_city', 'Неоновый город', 'background', 'uncommon', 400,
        '{"type": "cosmetic"}'::jsonb, true, '🖼️'),
    ('armor_vit', 'Доспех Стража', 'armor', 'rare', 900,
        '{"type": "xp_mult", "stat": "VIT", "value": 0.08}'::jsonb, true, '🛡️'),
    ('streak_freeze', 'Заморозка стрика', 'consumable', 'common', 300,
        '{"type": "streak_shield", "charges": 1}'::jsonb, true, '🧊'),

    -- Легендарные награды квестов/ачивок: не продаются (price NULL, purchasable=false)
    ('armor_titan', 'Доспех Титана', 'armor', 'legendary', NULL,
        '{"type": "streak_shield", "charges": 1}'::jsonb, false, '🛡️'),
    ('legendary_cape_conquistador', 'Плащ Конкистадора', 'aura', 'legendary', NULL,
        '{"type": "cosmetic"}'::jsonb, false, '🎯')
ON CONFLICT (id) DO NOTHING;

COMMIT;
