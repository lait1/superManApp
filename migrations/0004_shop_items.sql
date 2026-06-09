-- 0004_shop_items.sql — расширение каталога магазина (этап 4 графики).
-- Каждый не-consumable id имеет paper-doll спрайт в cmd/genassets (itemCatalog);
-- канон списка — internal/store/memory/seed.go (seedShopItems).
-- Идемпотентна: ON CONFLICT DO NOTHING.

BEGIN;

INSERT INTO shop_items (id, name, slot, rarity, price, effect, purchasable, icon) VALUES
    -- head
    ('helm_iron',   'Железный шлем',      'head',   'common',   250,
        '{"type": "cosmetic"}'::jsonb, true, '🪖'),
    ('hood_seeker', 'Капюшон Искателя',   'head',   'uncommon', 550,
        '{"type": "cosmetic"}'::jsonb, true, '🥷'),
    ('crown_sage',  'Корона Мудреца',     'head',   'epic',     4200,
        '{"type": "xp_mult", "stat": "INT", "value": 0.08}'::jsonb, true, '👑'),
    -- armor
    ('vest_padded', 'Стёганый жилет',     'armor',  'common',   300,
        '{"type": "cosmetic"}'::jsonb, true, '🦺'),
    -- weapon
    ('sword_short', 'Короткий меч',       'weapon', 'common',   250,
        '{"type": "cosmetic"}'::jsonb, true, '🗡️'),
    ('staff_arcane','Чародейский посох',  'weapon', 'rare',     1500,
        '{"type": "xp_mult", "stat": "INT", "value": 0.05}'::jsonb, true, '🪄'),
    -- back
    ('cloak_traveler', 'Плащ Путника',    'back',   'uncommon', 600,
        '{"type": "cosmetic"}'::jsonb, true, '🧣'),
    -- boots
    ('boots_leather',  'Кожаные сапоги',  'boots',  'common',   200,
        '{"type": "cosmetic"}'::jsonb, true, '👢'),
    ('boots_swift',    'Сапоги Ветра',    'boots',  'rare',     1100,
        '{"type": "xp_mult", "stat": "STR", "value": 0.05}'::jsonb, true, '👟'),
    -- легендарки (дроп/квест, не продаются)
    ('amulet_sun',     'Амулет Солнца',   'amulet', 'legendary', NULL,
        '{"type": "xp_mult", "stat": "CHA", "value": 0.10}'::jsonb, false, '☀️'),
    ('wings_phoenix',  'Крылья Феникса',  'back',   'legendary', NULL,
        '{"type": "cosmetic"}'::jsonb, false, '🪽')
ON CONFLICT (id) DO NOTHING;

COMMIT;
