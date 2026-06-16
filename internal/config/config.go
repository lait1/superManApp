// Package config holds runtime configuration loaded from environment variables
// and the tunable game-balance constants described in docs/03-progression-and-stats.md §1.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config carries process-level configuration read from the environment.
// See docs/07-architecture.md §5 for the source-of-truth list of env variables.
type Config struct {
	// Port is the HTTP listen port (env PORT, default 8080).
	Port string
	// DatabaseURL is the PostgreSQL connection string (env DATABASE_URL).
	// When empty the in-memory store is used.
	DatabaseURL string
	// TelegramBotToken is the bot token used for initData validation and sending
	// messages (env TELEGRAM_BOT_TOKEN).
	TelegramBotToken string
	// TelegramWebappURL is the public HTTPS URL of the Mini App, used for
	// web_app buttons (env TELEGRAM_WEBAPP_URL).
	TelegramWebappURL string
	// Env is the runtime environment: "dev" or "prod" (env ENV). In "dev" the
	// X-Device-Id auth fallback is enabled.
	Env string
	// NotifyTick is how often the scheduler ticks (env NOTIFY_TICK, e.g. "5m").
	NotifyTick time.Duration
	// StartingGold is the gold balance given to newly created characters
	// (env STARTING_GOLD). Defaults to 10000 in dev — enough to test the shop
	// end-to-end — and 0 in prod.
	StartingGold int64
	// MaintenanceMode closes the app for everyone except the admin and makes the
	// scheduler send notifications only to the admin (env MAINTENANCE_MODE). Used
	// while debugging the live app.
	MaintenanceMode bool
	// AdminTelegramID is the Telegram user id of the owner (env ADMIN_TELEGRAM_ID).
	// It is the only account that bypasses MaintenanceMode and still receives
	// notifications. 0 means "no admin" (app fully closed while in maintenance).
	AdminTelegramID int64
}

// Load reads configuration from the environment, applying defaults.
func Load() Config {
	cfg := Config{
		Port:              getenv("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		TelegramBotToken:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramWebappURL: os.Getenv("TELEGRAM_WEBAPP_URL"),
		Env:               getenv("ENV", "dev"),
		NotifyTick:        5 * time.Minute,
	}
	if raw := os.Getenv("NOTIFY_TICK"); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil {
			cfg.NotifyTick = d
		}
	}
	if cfg.IsDev() {
		cfg.StartingGold = 10000
	}
	if raw := os.Getenv("STARTING_GOLD"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil && v >= 0 {
			cfg.StartingGold = v
		}
	}
	if raw := os.Getenv("MAINTENANCE_MODE"); raw != "" {
		if v, err := strconv.ParseBool(raw); err == nil {
			cfg.MaintenanceMode = v
		}
	}
	if raw := os.Getenv("ADMIN_TELEGRAM_ID"); raw != "" {
		if v, err := strconv.ParseInt(raw, 10, 64); err == nil {
			cfg.AdminTelegramID = v
		}
	}
	return cfg
}

// IsDev reports whether the process runs in the development environment.
func (c Config) IsDev() bool { return c.Env == "dev" }

// IsAdmin reports whether the given Telegram user id is the configured owner.
// It is the single source of truth for the maintenance bypass and the
// admin-only notification rule. A zero AdminTelegramID matches nobody.
func (c Config) IsAdmin(tgID *int64) bool {
	return tgID != nil && *tgID != 0 && *tgID == c.AdminTelegramID
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// StreakTier maps a lower bound of consecutive days to an XP multiplier.
// See docs/03-progression-and-stats.md §1 STREAK_TIERS.
type StreakTier struct {
	Days int     `json:"days"`
	Mult float64 `json:"mult"`
}

// Balance holds the tunable game-balance constants from
// docs/03-progression-and-stats.md §1. These are meant to live in the DB/config
// and can be changed without a release; DefaultBalance provides the seed values.
type Balance struct {
	// XPBase is the base of the character-level curve (XP_BASE).
	XPBase float64
	// XPExp is the exponent of the character-level curve (XP_EXP).
	XPExp float64
	// StatBase is the base of the stat-level curve (STAT_BASE).
	StatBase float64
	// StatExp is the exponent of the stat-level curve (STAT_EXP).
	StatExp float64
	// CritChance is the probability of an XP crit (CRIT_CHANCE).
	CritChance float64
	// CritMult is the XP multiplier applied on a crit (CRIT_MULT).
	CritMult float64
	// StreakTiers is the ordered list of streak day → multiplier tiers.
	StreakTiers []StreakTier
	// GoldStreakBonus is the maximum extra gold multiplier from the streak
	// (GOLD_STREAK_BONUS).
	GoldStreakBonus float64
}

// DefaultBalance returns the seed balance values from
// docs/03-progression-and-stats.md §1.
func DefaultBalance() Balance {
	return Balance{
		XPBase:     100,
		XPExp:      1.5,
		StatBase:   60,
		StatExp:    1.4,
		CritChance: 0.10,
		CritMult:   2.0,
		StreakTiers: []StreakTier{
			{Days: 0, Mult: 1.0},
			{Days: 3, Mult: 1.1},
			{Days: 7, Mult: 1.25},
			{Days: 14, Mult: 1.4},
			{Days: 30, Mult: 1.5},
		},
		GoldStreakBonus: 0.5,
	}
}
