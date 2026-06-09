// Package domain defines the core entity types of superMen. Field shapes and
// JSON tags follow docs/08-data-model.md (storage) and docs/09-api.md (wire).
package domain

import "time"

// StatKey identifies one of the five RPG attributes (docs/01 §1).
type StatKey string

// The five stat keys.
const (
	StatSTR StatKey = "STR" // Сила  — body/sport
	StatINT StatKey = "INT" // Интеллект — knowledge/languages
	StatDIS StatKey = "DIS" // Дисциплина — work/routine
	StatVIT StatKey = "VIT" // Жизненная сила — health/recovery
	StatCHA StatKey = "CHA" // Харизма — social/new experience
)

// AllStatKeys is the canonical ordered set of stat keys (5 rows per character).
var AllStatKeys = []StatKey{StatSTR, StatINT, StatDIS, StatVIT, StatCHA}

// NotifPrefs are the per-user notification toggles (docs/06 §6).
type NotifPrefs struct {
	Daily          bool `json:"daily"`
	StreakReminder bool `json:"streakReminder"`
	Morning        bool `json:"morning"`
	Milestone      bool `json:"milestone"`
	DailyHour      int  `json:"dailyHour"`
}

// User mirrors the users table (docs/08 §2).
type User struct {
	ID             int64      `json:"id"`
	TelegramUserID *int64     `json:"telegramUserId,omitempty"`
	DeviceID       *string    `json:"deviceId,omitempty"`
	Username       string     `json:"username"`
	Timezone       string     `json:"timezone"`
	NotifPrefs     NotifPrefs `json:"notifPrefs"`
	CreatedAt      time.Time  `json:"createdAt"`
	LastSeenAt     *time.Time `json:"lastSeenAt,omitempty"`
}

// Appearance is the player-chosen visual customization of a character,
// selected during onboarding (docs/12). Values reference ids from the
// character asset manifest (bodyTypes / skinTones / hairstyles / hairColors).
type Appearance struct {
	BodyType  string `json:"bodyType"`
	SkinTone  string `json:"skinTone"`
	Hairstyle string `json:"hairstyle"`
	HairColor string `json:"hairColor"`
}

// Allowed appearance ids — the validation source of truth, kept in sync with
// the asset generator (cmd/genassets) and manifest.json.
var (
	BodyTypes  = []string{"a", "b"}
	SkinTones  = []string{"s1", "s2", "s3", "s4"}
	Hairstyles = []string{"bald", "short", "spiky", "long", "ponytail"}
	HairColors = []string{"dark", "brown", "blond", "red"}
)

// DefaultAppearance is the look of a character before onboarding completes.
func DefaultAppearance() Appearance {
	return Appearance{BodyType: "a", SkinTone: "s2", Hairstyle: "short", HairColor: "dark"}
}

// Character mirrors the characters table (docs/08 §2).
type Character struct {
	ID              int64            `json:"id"`
	UserID          int64            `json:"userId"`
	Name            string           `json:"name"`
	Level           int              `json:"level"`
	XPTotal         int64            `json:"xpTotal"`
	Gold            int64            `json:"gold"`
	Class           string           `json:"class"`
	Rank            string           `json:"rank"`
	StreakDays      int              `json:"streakDays"`
	BestStreak      int              `json:"bestStreak"`
	LastCheckinDate *time.Time       `json:"lastCheckinDate,omitempty"`
	Equipped        map[string]int64 `json:"equipped"`
	Appearance      Appearance       `json:"appearance"`
	// Onboarded reports whether the user completed first-run onboarding
	// (named the hero and picked an appearance).
	Onboarded bool `json:"onboarded"`
}

// Stat mirrors a stats row (docs/08 §2). Value accumulates points; Level is the
// derived stat level.
type Stat struct {
	CharacterID int64   `json:"-"`
	Key         StatKey `json:"key"`
	Value       int64   `json:"value"`
	Level       int     `json:"level"`
}

// Activity mirrors the activities catalog (docs/08 §3).
type Activity struct {
	Key         string  `json:"key"`
	Title       string  `json:"title"`
	StatKey     StatKey `json:"statKey"`
	BaseXP      int     `json:"baseXp"`
	BaseGold    int     `json:"baseGold"`
	HasDuration bool    `json:"hasDuration"`
	RefMinutes  int     `json:"refMinutes"`
	Rarity      string  `json:"rarity"`
	Icon        string  `json:"icon"`
	DailyCap    int     `json:"dailyCap"`
}

// Quest mirrors the quests catalog (docs/08 §3).
type Quest struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Type        string         `json:"type"` // daily|weekly|chain|side|class|balance
	Description string         `json:"description"`
	Condition   map[string]any `json:"condition"`
	Reward      QuestReward    `json:"reward"`
	Icon        string         `json:"icon"`
	Active      bool           `json:"active"`
}

// QuestReward is the reward payload of a quest (docs/09 GET /quests).
type QuestReward struct {
	XP    int    `json:"xp,omitempty"`
	Gold  int    `json:"gold,omitempty"`
	Title string `json:"title,omitempty"`
	Item  string `json:"item,omitempty"`
}

// QuestProgress mirrors the quest_progress table (docs/08 §2).
type QuestProgress struct {
	ID          int64      `json:"-"`
	CharacterID int64      `json:"-"`
	QuestID     string     `json:"id"`
	Progress    int        `json:"progress"`
	Target      int        `json:"target"`
	Status      string     `json:"status"` // active|completed|claimed|expired
	PeriodKey   string     `json:"periodKey,omitempty"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// ActivityLog mirrors an activity_logs row (docs/08 §2): one check-in record.
type ActivityLog struct {
	ID          int64     `json:"id"`
	CharacterID int64     `json:"-"`
	ActivityKey string    `json:"activityKey"`
	DurationMin int       `json:"durationMin,omitempty"`
	Note        string    `json:"note,omitempty"`
	XPAwarded   int       `json:"xpAwarded"`
	GoldAwarded int       `json:"goldAwarded"`
	StatAwarded int       `json:"statAwarded"`
	IsCrit      bool      `json:"isCrit"`
	CreatedAt   time.Time `json:"createdAt"`
	LocalDate   time.Time `json:"localDate"`
}

// QuestWithProgress joins a catalog quest with a character's progress, as
// returned to the quests endpoint (docs/09 GET /quests).
type QuestWithProgress struct {
	ID       string      `json:"id"`
	Title    string      `json:"title"`
	Type     string      `json:"type"`
	Progress int         `json:"progress"`
	Target   int         `json:"target"`
	Status   string      `json:"status"`
	Reward   QuestReward `json:"reward"`
	// Condition is the quest's raw condition (docs/08 §3). Not serialized to the
	// API; carried internally so the engine can match a check-in's activity
	// against the quest before advancing it.
	Condition map[string]any `json:"-"`
}

// AchievementWithState joins a catalog achievement with its unlock state.
type AchievementWithState struct {
	Achievement
	Unlocked   bool       `json:"unlocked"`
	UnlockedAt *time.Time `json:"unlockedAt,omitempty"`
}

// DailyReportView is the in-app daily summary (docs/06 §3, docs/09 GET /report/today).
type DailyReportView struct {
	Date        time.Time           `json:"date"`
	Entries     []ReportEntry       `json:"entries"`
	TotalXP     int                 `json:"totalXp"`
	TotalGold   int                 `json:"totalGold"`
	StreakDays  int                 `json:"streakDays"`
	StreakMult  float64             `json:"streakMult"`
	Level       int                 `json:"level"`
	XPIntoLevel int64               `json:"xpIntoLevel"`
	XPToNext    int64               `json:"xpToNext"`
	OpenQuests  []QuestWithProgress `json:"openQuests"`
	HadActivity bool                `json:"hadActivity"`
}

// ReportEntry is a single per-activity line in the daily report.
type ReportEntry struct {
	ActivityKey string  `json:"activityKey"`
	Title       string  `json:"title"`
	XP          int     `json:"xp"`
	StatKey     StatKey `json:"statKey"`
	IsCrit      bool    `json:"isCrit"`
	Count       int     `json:"count"`
}

// Achievement mirrors the achievements catalog (docs/08 §3).
type Achievement struct {
	ID          string         `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Category    string         `json:"category"`
	Condition   map[string]any `json:"condition"`
	Reward      map[string]any `json:"reward,omitempty"`
	Icon        string         `json:"icon"`
}

// AchievementUnlock mirrors an achievement_unlocks row (docs/08 §2).
type AchievementUnlock struct {
	CharacterID   int64     `json:"-"`
	AchievementID string    `json:"achievementId"`
	UnlockedAt    time.Time `json:"unlockedAt"`
}

// ItemEffect is the flexible effect payload of a shop item (docs/04 §5).
type ItemEffect struct {
	Type    string  `json:"type,omitempty"`
	Stat    StatKey `json:"stat,omitempty"`
	Value   float64 `json:"value,omitempty"`
	Charges int     `json:"charges,omitempty"`
}

// ShopItem mirrors the shop_items catalog (docs/08 §3).
type ShopItem struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Slot        string     `json:"slot"` // weapon|armor|amulet|aura|background|consumable
	Rarity      string     `json:"rarity"`
	Price       *int       `json:"price"` // nil = not for sale → serialized as explicit null (matches client type number|null)
	Effect      ItemEffect `json:"effect"`
	Purchasable bool       `json:"purchasable"`
	Icon        string     `json:"icon"`
}

// InventoryItem mirrors the inventory_items table (docs/08 §2).
type InventoryItem struct {
	ID          int64     `json:"id"`
	CharacterID int64     `json:"-"`
	ShopItemID  string    `json:"shopItemId"`
	AcquiredVia string    `json:"acquiredVia"` // purchase|drop|quest|achievement
	Quantity    int       `json:"quantity"`
	AcquiredAt  time.Time `json:"acquiredAt"`
}

// Transaction mirrors the transactions gold ledger (docs/08 §2).
type Transaction struct {
	ID          int64     `json:"id"`
	CharacterID int64     `json:"-"`
	Amount      int       `json:"amount"` // + credit / - debit
	Reason      string    `json:"reason"` // checkin|quest|achievement|purchase|levelup
	RefID       string    `json:"refId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

// DailyReport mirrors the daily_reports idempotency table (docs/08 §2).
type DailyReport struct {
	ID         int64          `json:"id"`
	UserID     int64          `json:"userId"`
	ReportDate time.Time      `json:"reportDate"`
	Kind       string         `json:"kind"` // daily|streak_reminder|morning|milestone
	SentAt     time.Time      `json:"sentAt"`
	Payload    map[string]any `json:"payload,omitempty"`
}

// LevelChange describes a from→to transition (level, rank or stat level).
type LevelChange struct {
	From int `json:"from"`
	To   int `json:"to"`
}

// RankChange describes a from→to rank transition.
type RankChange struct {
	From string `json:"from"`
	To   string `json:"to"`
}

// StatLevelChange describes a stat-level transition for a specific stat.
type StatLevelChange struct {
	Key  StatKey `json:"key"`
	From int     `json:"from"`
	To   int     `json:"to"`
}

// Drop describes an item that dropped after a check-in (docs/09 POST /checkin).
type Drop struct {
	ItemID string `json:"itemId"`
	Name   string `json:"name"`
	Rarity string `json:"rarity"`
	Slot   string `json:"slot"`
}

// CharacterSummary is the compact character snapshot returned in reward events.
type CharacterSummary struct {
	Level      int   `json:"level"`
	XPTotal    int64 `json:"xpTotal"`
	Gold       int64 `json:"gold"`
	StreakDays int   `json:"streakDays"`
}

// RewardCore holds the scalar reward of a check-in. docs/09 POST /checkin
// nests these fields under a "reward" object, separate from the side-events
// (drop, levelUp, ...).
type RewardCore struct {
	XP         int     `json:"xp"`
	Gold       int     `json:"gold"`
	StatKey    StatKey `json:"statKey"`
	StatPoints int     `json:"statPoints"`
	IsCrit     bool    `json:"isCrit"`
	StreakDays int     `json:"streakDays"`
	StreakMult float64 `json:"streakMult"`
}

// RewardEvent is the result of a check-in. Field names and JSON tags follow
// docs/09 POST /checkin exactly: the scalar reward is nested under "reward",
// while optional side-events are nil/empty when absent.
type RewardEvent struct {
	Reward               RewardCore       `json:"reward"`
	Drop                 *Drop            `json:"drop"`
	LevelUp              *LevelChange     `json:"levelUp"`
	RankUp               *RankChange      `json:"rankUp"`
	StatLevelUp          *StatLevelChange `json:"statLevelUp"`
	QuestsAdvanced       []QuestProgress  `json:"questsAdvanced"`
	AchievementsUnlocked []string         `json:"achievementsUnlocked"`
	Character            CharacterSummary `json:"character"`
}
