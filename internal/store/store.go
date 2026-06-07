// Package store defines the persistence contract used by the game engine, the
// API handlers and the scheduler. Implementations live in store/memory and
// store/postgres.
package store

import (
	"context"
	"errors"
	"time"

	"superMen/internal/domain"
)

// Common sentinel errors returned by Store implementations.
var (
	// ErrNotFound is returned when a requested entity does not exist.
	ErrNotFound = errors.New("not found")
	// ErrInsufficientGold is returned by BuyItem when the character cannot
	// afford an item (maps to HTTP 409, code "insufficient_gold").
	ErrInsufficientGold = errors.New("insufficient gold")
	// ErrNotImplemented marks a skeleton method that has no behaviour yet.
	ErrNotImplemented = errors.New("not implemented")
)

// Store is the full persistence interface. All methods take a context.Context
// as the first argument. Methods are grouped by domain area.
type Store interface {
	// --- Users & identity (docs/10) ---

	// GetOrCreateUserByTelegramID finds or creates a user (plus character and
	// the 5 stat rows) keyed by Telegram user id.
	GetOrCreateUserByTelegramID(ctx context.Context, telegramUserID int64, username string) (*domain.User, error)
	// GetOrCreateUserByDeviceID finds or creates a user keyed by a dev device id.
	GetOrCreateUserByDeviceID(ctx context.Context, deviceID string) (*domain.User, error)

	// --- Character & stats ---

	// GetCharacter returns the character belonging to a user.
	GetCharacter(ctx context.Context, userID int64) (*domain.Character, error)
	// SaveCharacter persists denormalized character fields (level/gold/streak/etc).
	SaveCharacter(ctx context.Context, ch *domain.Character) error
	// GetStats returns the 5 stat rows of a character.
	GetStats(ctx context.Context, characterID int64) ([]domain.Stat, error)
	// SaveStat upserts a single stat row.
	SaveStat(ctx context.Context, st *domain.Stat) error

	// --- Activities & check-ins ---

	// ListActivities returns the activities catalog.
	ListActivities(ctx context.Context) ([]domain.Activity, error)
	// GetActivity returns a single activity by key.
	GetActivity(ctx context.Context, key string) (*domain.Activity, error)
	// TodayCheckins returns the activity keys a character checked in on a given
	// local date (used for the daily cap and the /me screen).
	TodayCheckins(ctx context.Context, characterID int64, localDate time.Time) ([]string, error)
	// InsertActivityLog records one check-in in the audit log.
	InsertActivityLog(ctx context.Context, log *domain.ActivityLog) error

	// --- Quests ---

	// ListQuestsWithProgress returns active quests with the character's progress.
	ListQuestsWithProgress(ctx context.Context, characterID int64) ([]domain.QuestWithProgress, error)
	// UpsertQuestProgress creates or updates a quest_progress row.
	UpsertQuestProgress(ctx context.Context, qp *domain.QuestProgress) error
	// ClaimQuest marks a completed quest as claimed and returns its reward.
	ClaimQuest(ctx context.Context, characterID int64, questID string) (*domain.QuestReward, error)

	// --- Achievements ---

	// ListAchievements returns all achievements with unlock state for a character.
	ListAchievements(ctx context.Context, characterID int64) ([]domain.AchievementWithState, error)
	// UnlockAchievement records an achievement unlock (idempotent).
	UnlockAchievement(ctx context.Context, characterID int64, achievementID string) error

	// --- Shop & inventory ---

	// ListShopItems returns the shop catalog.
	ListShopItems(ctx context.Context) ([]domain.ShopItem, error)
	// BuyItem debits gold and adds the item to the inventory, returning the new
	// gold balance and inventory item id. Returns ErrInsufficientGold on failure.
	BuyItem(ctx context.Context, characterID int64, itemID string) (newGold int64, inventoryItemID int64, err error)
	// ListInventory returns a character's inventory items.
	ListInventory(ctx context.Context, characterID int64) ([]domain.InventoryItem, error)
	// EquipItem equips an inventory item into its slot and returns the new
	// equipped map.
	EquipItem(ctx context.Context, characterID int64, inventoryItemID int64) (equipped map[string]int64, err error)
	// UnequipItem removes an inventory item from its slot and returns the new
	// equipped map.
	UnequipItem(ctx context.Context, characterID int64, inventoryItemID int64) (equipped map[string]int64, err error)

	// --- Transactions ---

	// AddTransaction appends a row to the gold ledger.
	AddTransaction(ctx context.Context, tx *domain.Transaction) error

	// --- Reports & notifications ---

	// GetReportToday builds the in-app daily report for a character/date.
	GetReportToday(ctx context.Context, characterID int64, localDate time.Time) (*domain.DailyReportView, error)
	// MarkReportSent records that a notification of a kind was sent for a date
	// (idempotency). Returns false if it was already recorded.
	MarkReportSent(ctx context.Context, userID int64, reportDate time.Time, kind string) (bool, error)
	// UsersForNotificationSlot returns users whose local time matches the given
	// slot hour and who have not yet been notified of kind today.
	UsersForNotificationSlot(ctx context.Context, now time.Time, slotHour int, kind string) ([]domain.User, error)
	// UpdateNotificationSettings updates a user's timezone and notification prefs.
	UpdateNotificationSettings(ctx context.Context, userID int64, tz string, prefs domain.NotifPrefs) error
}
