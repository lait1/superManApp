package api

import "superMen/internal/domain"

// MeResponse is the GET /api/v1/me payload (docs/09 §3).
type MeResponse struct {
	Character CharacterDTO `json:"character"`
	Stats     []StatDTO    `json:"stats"`
	// TodayCheckins lists the activity keys checked in today.
	TodayCheckins []string `json:"todayCheckins"`
}

// CharacterDTO is the character block of GET /me (docs/09 §3).
type CharacterDTO struct {
	Name        string            `json:"name"`
	Level       int               `json:"level"`
	XPTotal     int64             `json:"xpTotal"`
	XPToNext    int64             `json:"xpToNext"`
	XPIntoLevel int64             `json:"xpIntoLevel"`
	Gold        int64             `json:"gold"`
	Class       string            `json:"class"`
	Rank        string            `json:"rank"`
	StreakDays  int               `json:"streakDays"`
	StreakMult  float64           `json:"streakMult"`
	Equipped    map[string]int64  `json:"equipped"`
	Appearance  domain.Appearance `json:"appearance"`
	Onboarded   bool              `json:"onboarded"`
}

// CharacterSetupRequest is the POST /api/v1/character/setup body: the
// onboarding step where the user names the hero and picks an appearance.
type CharacterSetupRequest struct {
	Name       string            `json:"name"`
	Appearance domain.Appearance `json:"appearance"`
}

// CharacterSetupResponse is the POST /api/v1/character/setup payload.
type CharacterSetupResponse struct {
	OK        bool         `json:"ok"`
	Character CharacterDTO `json:"character"`
}

// StatDTO is one stat row of GET /me (docs/09 §3).
type StatDTO struct {
	Key       domain.StatKey `json:"key"`
	Value     int64          `json:"value"`
	Level     int            `json:"level"`
	IntoLevel int64          `json:"intoLevel"`
	ToNext    int64          `json:"toNext"`
}

// CheckinRequest is the POST /api/v1/checkin body (docs/09 §3).
type CheckinRequest struct {
	ActivityKey string `json:"activityKey"`
	DurationMin int    `json:"durationMin"`
	Note        string `json:"note"`
}

// CheckinResponse is the POST /api/v1/checkin reward event. It is the
// domain.RewardEvent, whose JSON tags match docs/09 §3 exactly.
type CheckinResponse = domain.RewardEvent

// ActivitiesResponse is the GET /api/v1/activities payload.
type ActivitiesResponse struct {
	Activities []domain.Activity `json:"activities"`
}

// QuestsResponse is the GET /api/v1/quests payload (docs/09 §3).
type QuestsResponse struct {
	Daily  []domain.QuestWithProgress `json:"daily"`
	Weekly []domain.QuestWithProgress `json:"weekly"`
	Chains []domain.QuestWithProgress `json:"chains"`
}

// ClaimResponse is the POST /api/v1/quests/{id}/claim payload.
type ClaimResponse struct {
	OK     bool               `json:"ok"`
	Reward domain.QuestReward `json:"reward"`
}

// AchievementsResponse is the GET /api/v1/achievements payload.
type AchievementsResponse struct {
	Achievements []domain.AchievementWithState `json:"achievements"`
}

// ShopResponse is the GET /api/v1/shop payload.
type ShopResponse struct {
	Items []domain.ShopItem `json:"items"`
}

// BuyResponse is the POST /api/v1/shop/{itemId}/buy success payload (docs/09 §3).
type BuyResponse struct {
	OK              bool  `json:"ok"`
	Gold            int64 `json:"gold"`
	InventoryItemID int64 `json:"inventoryItemId"`
}

// InventoryItemDTO is one row of GET /api/v1/inventory: the inventory record
// joined with its shop-catalog entry (name/slot/rarity/icon) plus the live
// equipped state — exactly what the client inventory list renders.
type InventoryItemDTO struct {
	ID          int64  `json:"id"`
	ShopItemID  string `json:"shopItemId"`
	Name        string `json:"name"`
	Slot        string `json:"slot"`
	Rarity      string `json:"rarity"`
	AcquiredVia string `json:"acquiredVia"`
	Quantity    int    `json:"quantity"`
	Icon        string `json:"icon,omitempty"`
	Equipped    bool   `json:"equipped"`
}

// EquipResponse is the POST /api/v1/inventory/{id}/equip payload (docs/09 §3).
type EquipResponse struct {
	OK           bool              `json:"ok"`
	Equipped     map[string]int64  `json:"equipped"`
	StatsPreview map[string]string `json:"statsPreview,omitempty"`
}

// ReportResponse is the GET /api/v1/report/today payload (docs/06 §3).
type ReportResponse struct {
	Report domain.DailyReportView `json:"report"`
}

// NotificationSettingsRequest is the PUT /api/v1/settings/notifications body
// (docs/09 §3).
type NotificationSettingsRequest struct {
	Timezone       string `json:"timezone"`
	Daily          bool   `json:"daily"`
	StreakReminder bool   `json:"streakReminder"`
	Morning        bool   `json:"morning"`
	Milestone      bool   `json:"milestone"`
	DailyHour      int    `json:"dailyHour"`
}

// ErrorBody is the inner object of the unified error envelope (docs/09 §1).
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// ErrorResponse is the unified error envelope: {"error": {"code","message"}}.
type ErrorResponse struct {
	Error ErrorBody `json:"error"`
}
