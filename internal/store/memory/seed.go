package memory

import "superMen/internal/domain"

// seedCatalogs populates the read-only catalog tables (activities, quests,
// achievements, shop_items) used by the offline demo. Values follow the
// starter catalogs in docs/02-activities-and-quests.md and
// docs/04-economy-and-shop.md. The caller (New) holds no lock since this runs
// before the Store is shared.
func (s *Store) seedCatalogs() {
	s.seedActivities()
	s.seedQuests()
	s.seedAchievements()
	s.seedShopItems()
}

func (s *Store) seedActivities() {
	s.activities = []domain.Activity{
		{Key: "gym", Title: "Спортзал", StatKey: domain.StatSTR, BaseXP: 50, BaseGold: 30, HasDuration: true, RefMinutes: 60, Rarity: "common", Icon: "🏋️", DailyCap: 1},
		{Key: "cardio", Title: "Кардио / бег", StatKey: domain.StatSTR, BaseXP: 40, BaseGold: 25, HasDuration: true, RefMinutes: 30, Rarity: "common", Icon: "🏃", DailyCap: 1},
		{Key: "english", Title: "Английский", StatKey: domain.StatINT, BaseXP: 40, BaseGold: 20, HasDuration: true, RefMinutes: 30, Rarity: "common", Icon: "🧠", DailyCap: 1},
		{Key: "spanish", Title: "Испанский", StatKey: domain.StatINT, BaseXP: 40, BaseGold: 20, HasDuration: true, RefMinutes: 30, Rarity: "common", Icon: "🗣️", DailyCap: 1},
		{Key: "reading", Title: "Чтение", StatKey: domain.StatINT, BaseXP: 25, BaseGold: 15, HasDuration: true, RefMinutes: 30, Rarity: "common", Icon: "📖", DailyCap: 1},
		{Key: "work", Title: "Рабочая фокус-сессия", StatKey: domain.StatDIS, BaseXP: 35, BaseGold: 20, HasDuration: true, RefMinutes: 50, Rarity: "common", Icon: "💼", DailyCap: 2},
		{Key: "early_rise", Title: "Ранний подъём", StatKey: domain.StatDIS, BaseXP: 20, BaseGold: 10, HasDuration: false, Rarity: "common", Icon: "⏰", DailyCap: 1},
		{Key: "meditation", Title: "Медитация", StatKey: domain.StatVIT, BaseXP: 25, BaseGold: 15, HasDuration: true, RefMinutes: 15, Rarity: "common", Icon: "🧘", DailyCap: 1},
		{Key: "sleep", Title: "Здоровый сон", StatKey: domain.StatVIT, BaseXP: 20, BaseGold: 10, HasDuration: false, Rarity: "common", Icon: "😴", DailyCap: 1},
		{Key: "nutrition", Title: "Здоровое питание", StatKey: domain.StatVIT, BaseXP: 20, BaseGold: 10, HasDuration: false, Rarity: "common", Icon: "🥗", DailyCap: 1},
		{Key: "adventure", Title: "Приключение / новый опыт", StatKey: domain.StatCHA, BaseXP: 70, BaseGold: 50, HasDuration: false, Rarity: "rare", Icon: "🧭", DailyCap: 1},
		{Key: "networking", Title: "Встреча / нетворкинг", StatKey: domain.StatCHA, BaseXP: 35, BaseGold: 20, HasDuration: false, Rarity: "common", Icon: "🤝", DailyCap: 1},
		{Key: "speech", Title: "Публичное выступление", StatKey: domain.StatCHA, BaseXP: 80, BaseGold: 60, HasDuration: false, Rarity: "rare", Icon: "🎤", DailyCap: 1},
	}
	s.activityByKey = make(map[string]domain.Activity, len(s.activities))
	for _, a := range s.activities {
		s.activityByKey[a.Key] = a
	}
}

func (s *Store) seedQuests() {
	s.quests = []domain.Quest{
		{
			ID: "daily_english_30", Title: "30 мин английского", Type: "daily",
			Description: "Сделай 30 минут английского сегодня",
			Condition:   map[string]any{"activity": "english", "minutes": 30},
			Reward:      domain.QuestReward{XP: 60, Gold: 25},
			Icon:        "🧠", Active: true,
		},
		{
			ID: "daily_work_2", Title: "2 фокус-сессии", Type: "daily",
			Description: "Проведи 2 рабочие фокус-сессии",
			Condition:   map[string]any{"activity": "work", "target": 2},
			Reward:      domain.QuestReward{XP: 100, Gold: 50},
			Icon:        "💼", Active: true,
		},
		{
			ID: "weekly_gym_3", Title: "3 тренировки", Type: "weekly",
			Description: "Сходи в спортзал 3 раза за неделю",
			Condition:   map[string]any{"activity": "gym", "target": 3},
			Reward:      domain.QuestReward{XP: 300, Gold: 150},
			Icon:        "🏋️", Active: true,
		},
		{
			ID: "weekly_lang_5", Title: "5 языковых занятий", Type: "weekly",
			Description: "5 занятий любым языком за неделю",
			Condition:   map[string]any{"activity": "language", "target": 5},
			Reward:      domain.QuestReward{XP: 300, Gold: 150},
			Icon:        "🗣️", Active: true,
		},
		{
			ID: "chain_polyglot", Title: "Полиглот", Type: "chain",
			Description: "14 дней любого языка подряд",
			Condition:   map[string]any{"activity": "language", "streak_days": 14},
			Reward:      domain.QuestReward{XP: 700, Gold: 300, Title: "Полиглот"},
			Icon:        "🧠", Active: true,
		},
		{
			ID: "chain_conquistador", Title: "Конкистадор", Type: "chain",
			Description: "Занимайся испанским 30 дней подряд",
			Condition:   map[string]any{"activity": "spanish", "streak_days": 30},
			Reward:      domain.QuestReward{XP: 1000, Gold: 500, Title: "Conquistador", Item: "legendary_cape_conquistador"},
			Icon:        "🗣️", Active: true,
		},
		{
			ID: "chain_iron", Title: "Железный", Type: "chain",
			Description: "50 тренировок суммарно",
			Condition:   map[string]any{"activity": "gym", "count": 50},
			Reward:      domain.QuestReward{XP: 1000, Gold: 500, Item: "armor_titan"},
			Icon:        "🏋️", Active: true,
		},
		{
			ID: "chain_discipline", Title: "Несокрушимый", Type: "chain",
			Description: "21 день подряд с фокус-сессией",
			Condition:   map[string]any{"activity": "work", "streak_days": 21},
			Reward:      domain.QuestReward{XP: 800, Gold: 400, Title: "Несокрушимый"},
			Icon:        "💼", Active: true,
		},
		{
			ID: "side_discovery", Title: "Открытие", Type: "side",
			Description: "Попробуй что-то, чего никогда не делал",
			Condition:   map[string]any{"activity": "adventure", "target": 1},
			Reward:      domain.QuestReward{XP: 150, Gold: 100},
			Icon:        "🧭", Active: true,
		},
	}
	s.questByID = make(map[string]domain.Quest, len(s.quests))
	for _, q := range s.quests {
		s.questByID[q.ID] = q
	}
}

func (s *Store) seedAchievements() {
	s.achievements = []domain.Achievement{
		{ID: "first_checkin", Title: "Первый шаг", Description: "Первый чек-ин", Category: "start", Condition: map[string]any{"checkins": 1}, Reward: map[string]any{"xp": 50}, Icon: "👣"},
		{ID: "first_levelup", Title: "Рост", Description: "Первый level-up", Category: "start", Condition: map[string]any{"level": 2}, Reward: map[string]any{"gold": 50}, Icon: "⬆️"},
		{ID: "first_drop", Title: "Находка", Description: "Первый дроп шмота", Category: "start", Condition: map[string]any{"drops": 1}, Icon: "🎁"},
		{ID: "streak_7", Title: "Неделя огня", Description: "Стрик 7 дней", Category: "streak", Condition: map[string]any{"streak_days": 7}, Reward: map[string]any{"xp": 100, "gold": 50}, Icon: "🔥"},
		{ID: "streak_30", Title: "Месяц силы", Description: "Стрик 30 дней", Category: "streak", Condition: map[string]any{"streak_days": 30}, Reward: map[string]any{"xp": 500, "gold": 250}, Icon: "🔥"},
		{ID: "streak_100", Title: "Сотня", Description: "Стрик 100 дней", Category: "streak", Condition: map[string]any{"streak_days": 100}, Reward: map[string]any{"xp": 2000, "gold": 1000}, Icon: "🔥"},
		{ID: "volume_gym_100", Title: "100 тренировок", Description: "100 тренировок суммарно", Category: "volume", Condition: map[string]any{"activity": "gym", "count": 100}, Icon: "🏋️"},
		{ID: "level_10", Title: "LVL 10", Description: "Достигни 10 уровня", Category: "level", Condition: map[string]any{"level": 10}, Reward: map[string]any{"gold": 200}, Icon: "🎖️"},
		{ID: "level_25", Title: "LVL 25", Description: "Достигни 25 уровня", Category: "level", Condition: map[string]any{"level": 25}, Reward: map[string]any{"gold": 500}, Icon: "🎖️"},
		{ID: "level_50", Title: "LVL 50", Description: "Достигни 50 уровня", Category: "level", Condition: map[string]any{"level": 50}, Reward: map[string]any{"gold": 1000}, Icon: "🏅"},
		{ID: "level_100", Title: "LVL 100", Description: "Достигни 100 уровня", Category: "level", Condition: map[string]any{"level": 100}, Reward: map[string]any{"gold": 2500}, Icon: "👑"},
		{ID: "harmony", Title: "Гармония", Description: "Все статы ≥ LVL 10", Category: "balance", Condition: map[string]any{"all_stats_level": 10}, Reward: map[string]any{"title": "Гармония"}, Icon: "☯️"},
		{ID: "polyglot", Title: "Полиглот", Description: "Заверши цепочку «Полиглот»", Category: "domain", Condition: map[string]any{"quest": "chain_polyglot"}, Reward: map[string]any{"title": "Полиглот"}, Icon: "🧠"},
	}
	s.achByID = make(map[string]domain.Achievement, len(s.achievements))
	for _, a := range s.achievements {
		s.achByID[a.ID] = a
	}
}

func (s *Store) seedShopItems() {
	priceOwl := 1200
	priceBlade := 3500
	priceNeon := 400
	priceFreeze := 300
	priceVitArmor := 900

	s.shopItems = []domain.ShopItem{
		{ID: "amulet_owl", Name: "Амулет Совы", Slot: "amulet", Rarity: "rare", Price: &priceOwl, Effect: domain.ItemEffect{Type: "xp_mult", Stat: domain.StatINT, Value: 0.10}, Purchasable: true, Icon: "📿"},
		{ID: "blade_focus", Name: "Клинок Фокуса", Slot: "weapon", Rarity: "epic", Price: &priceBlade, Effect: domain.ItemEffect{Type: "xp_mult", Stat: domain.StatDIS, Value: 0.10}, Purchasable: true, Icon: "⚔️"},
		{ID: "bg_neon_city", Name: "Неоновый город", Slot: "background", Rarity: "uncommon", Price: &priceNeon, Effect: domain.ItemEffect{Type: "cosmetic"}, Purchasable: true, Icon: "🖼️"},
		{ID: "armor_vit", Name: "Доспех Стража", Slot: "armor", Rarity: "rare", Price: &priceVitArmor, Effect: domain.ItemEffect{Type: "xp_mult", Stat: domain.StatVIT, Value: 0.08}, Purchasable: true, Icon: "🛡️"},
		{ID: "streak_freeze", Name: "Заморозка стрика", Slot: "consumable", Rarity: "common", Price: &priceFreeze, Effect: domain.ItemEffect{Type: "streak_shield", Charges: 1}, Purchasable: true, Icon: "🧊"},
		// Legendary quest/achievement rewards: not purchasable (price nil).
		{ID: "armor_titan", Name: "Доспех Титана", Slot: "armor", Rarity: "legendary", Price: nil, Effect: domain.ItemEffect{Type: "streak_shield", Charges: 1}, Purchasable: false, Icon: "🛡️"},
		{ID: "legendary_cape_conquistador", Name: "Плащ Конкистадора", Slot: "aura", Rarity: "legendary", Price: nil, Effect: domain.ItemEffect{Type: "cosmetic"}, Purchasable: false, Icon: "🎯"},
	}
	s.shopByID = make(map[string]domain.ShopItem, len(s.shopItems))
	for _, it := range s.shopItems {
		s.shopByID[it.ID] = it
	}
}
