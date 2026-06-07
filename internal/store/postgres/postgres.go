// Package postgres provides a database/sql-backed store.Store implementation
// for PostgreSQL 16 (schema in docs/08-data-model.md).
//
// NOTE: this package intentionally does NOT import a concrete SQL driver. On
// deploy, register the pgx stdlib driver (a blank import of
// "github.com/jackc/pgx/v5/stdlib") in main/cmd and open the *sql.DB with
// sql.Open("pgx", DATABASE_URL); then pass it to NewStore. Keeping the driver
// out of this package lets it build with only the standard library.
//
// All queries are parameterized with PostgreSQL placeholders ($1, $2, ...).
// JSONB columns (notif_prefs, equipped, condition, reward, effect, payload) are
// marshaled/unmarshaled with encoding/json. The gold-mutating paths (BuyItem)
// and identity creation run inside a *sql.Tx so they are atomic.
package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"superMen/internal/domain"
	"superMen/internal/store"
)

// Store is a PostgreSQL-backed implementation of store.Store.
type Store struct {
	db *sql.DB
}

// NewStore constructs a Store over an already-open *sql.DB. The caller is
// responsible for opening the DB with the pgx stdlib driver (see package doc).
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// compile-time assertion that *Store satisfies store.Store.
var _ store.Store = (*Store)(nil)

// --- JSON helpers for JSONB columns ---

// marshalJSON encodes v to a []byte for a JSONB parameter; nil/empty becomes
// the JSON object literal "{}".
func marshalJSON(v any) ([]byte, error) {
	if v == nil {
		return []byte("{}"), nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return []byte("{}"), nil
	}
	return b, nil
}

// unmarshalJSON decodes a JSONB column (which arrives as []byte or string) into
// dst. Empty input is treated as the empty object.
func unmarshalJSON(raw any, dst any) error {
	var b []byte
	switch v := raw.(type) {
	case nil:
		return nil
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return nil
	}
	if len(b) == 0 {
		return nil
	}
	return json.Unmarshal(b, dst)
}

// --- Users & identity ---

func (s *Store) GetOrCreateUserByTelegramID(ctx context.Context, telegramUserID int64, username string) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := scanUser(tx.QueryRowContext(ctx,
		`SELECT id, telegram_user_id, device_id, username, timezone, notif_prefs, created_at, last_seen_at
		   FROM users WHERE telegram_user_id = $1`, telegramUserID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		// Existing user: refresh last_seen_at and username.
		if _, err := tx.ExecContext(ctx,
			`UPDATE users SET last_seen_at = now(), username = COALESCE(NULLIF($2,''), username) WHERE id = $1`,
			u.ID, username); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return u, nil
	}

	// Create user + character + 5 stats.
	prefs, _ := marshalJSON(defaultNotifPrefs())
	var userID int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO users (telegram_user_id, username, timezone, notif_prefs, last_seen_at)
		 VALUES ($1, $2, 'UTC', $3, now()) RETURNING id`,
		telegramUserID, username, prefs).Scan(&userID); err != nil {
		return nil, err
	}
	if err := createCharacterTx(ctx, tx, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCharacterOwnerUser(ctx, userID)
}

func (s *Store) GetOrCreateUserByDeviceID(ctx context.Context, deviceID string) (*domain.User, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	u, err := scanUser(tx.QueryRowContext(ctx,
		`SELECT id, telegram_user_id, device_id, username, timezone, notif_prefs, created_at, last_seen_at
		   FROM users WHERE device_id = $1`, deviceID))
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET last_seen_at = now() WHERE id = $1`, u.ID); err != nil {
			return nil, err
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return u, nil
	}

	prefs, _ := marshalJSON(defaultNotifPrefs())
	var userID int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO users (device_id, username, timezone, notif_prefs, last_seen_at)
		 VALUES ($1, 'guest', 'UTC', $2, now()) RETURNING id`,
		deviceID, prefs).Scan(&userID); err != nil {
		return nil, err
	}
	if err := createCharacterTx(ctx, tx, userID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.GetCharacterOwnerUser(ctx, userID)
}

// GetCharacterOwnerUser loads a user by id (used after creation).
func (s *Store) GetCharacterOwnerUser(ctx context.Context, userID int64) (*domain.User, error) {
	return scanUser(s.db.QueryRowContext(ctx,
		`SELECT id, telegram_user_id, device_id, username, timezone, notif_prefs, created_at, last_seen_at
		   FROM users WHERE id = $1`, userID))
}

// defaultNotifPrefs returns the default notification toggles for a new user.
func defaultNotifPrefs() domain.NotifPrefs {
	return domain.NotifPrefs{
		Daily:          true,
		StreakReminder: true,
		Morning:        false,
		Milestone:      true,
		DailyHour:      21,
	}
}

// createCharacterTx inserts a character row and the 5 stat rows for a user
// inside the given transaction.
func createCharacterTx(ctx context.Context, tx *sql.Tx, userID int64) error {
	var charID int64
	if err := tx.QueryRowContext(ctx,
		`INSERT INTO characters (user_id, name, equipped)
		 VALUES ($1, 'superMen', '{}') RETURNING id`, userID).Scan(&charID); err != nil {
		return err
	}
	for _, k := range domain.AllStatKeys {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO stats (character_id, stat_key, value, level) VALUES ($1, $2, 0, 1)`,
			charID, string(k)); err != nil {
			return err
		}
	}
	return nil
}

// rowScanner abstracts *sql.Row and *sql.Rows for shared scan helpers.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanUser scans a users row into a domain.User, decoding the notif_prefs JSONB.
func scanUser(row rowScanner) (*domain.User, error) {
	var (
		u       domain.User
		tgID    sql.NullInt64
		devID   sql.NullString
		uname   sql.NullString
		prefs   []byte
		lastSee sql.NullTime
	)
	if err := row.Scan(&u.ID, &tgID, &devID, &uname, &u.Timezone, &prefs, &u.CreatedAt, &lastSee); err != nil {
		return nil, err
	}
	if tgID.Valid {
		v := tgID.Int64
		u.TelegramUserID = &v
	}
	if devID.Valid {
		v := devID.String
		u.DeviceID = &v
	}
	u.Username = uname.String
	if lastSee.Valid {
		t := lastSee.Time
		u.LastSeenAt = &t
	}
	if err := unmarshalJSON(prefs, &u.NotifPrefs); err != nil {
		return nil, err
	}
	return &u, nil
}

// --- Character & stats ---

func (s *Store) GetCharacter(ctx context.Context, userID int64) (*domain.Character, error) {
	ch, err := scanCharacter(s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, level, xp_total, gold, class, rank, streak_days, best_streak, last_checkin_date, equipped
		   FROM characters WHERE user_id = $1`, userID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return ch, err
}

// scanCharacter scans a characters row, decoding the equipped JSONB.
func scanCharacter(row rowScanner) (*domain.Character, error) {
	var (
		ch       domain.Character
		lastDate sql.NullTime
		equipped []byte
	)
	if err := row.Scan(&ch.ID, &ch.UserID, &ch.Name, &ch.Level, &ch.XPTotal, &ch.Gold,
		&ch.Class, &ch.Rank, &ch.StreakDays, &ch.BestStreak, &lastDate, &equipped); err != nil {
		return nil, err
	}
	if lastDate.Valid {
		t := lastDate.Time
		ch.LastCheckinDate = &t
	}
	ch.Equipped = map[string]int64{}
	if err := unmarshalJSON(equipped, &ch.Equipped); err != nil {
		return nil, err
	}
	return &ch, nil
}

func (s *Store) SaveCharacter(ctx context.Context, ch *domain.Character) error {
	equipped, err := marshalJSON(ch.Equipped)
	if err != nil {
		return err
	}
	var lastDate any
	if ch.LastCheckinDate != nil {
		lastDate = *ch.LastCheckinDate
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE characters SET name=$2, level=$3, xp_total=$4, gold=$5, class=$6, rank=$7,
		        streak_days=$8, best_streak=$9, last_checkin_date=$10, equipped=$11
		 WHERE id=$1`,
		ch.ID, ch.Name, ch.Level, ch.XPTotal, ch.Gold, ch.Class, ch.Rank,
		ch.StreakDays, ch.BestStreak, lastDate, equipped)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}

func (s *Store) GetStats(ctx context.Context, characterID int64) ([]domain.Stat, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT character_id, stat_key, value, level FROM stats WHERE character_id = $1`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byKey := make(map[domain.StatKey]domain.Stat, 5)
	for rows.Next() {
		var st domain.Stat
		var key string
		if err := rows.Scan(&st.CharacterID, &key, &st.Value, &st.Level); err != nil {
			return nil, err
		}
		st.Key = domain.StatKey(key)
		byKey[st.Key] = st
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Return in canonical order.
	out := make([]domain.Stat, 0, len(domain.AllStatKeys))
	for _, k := range domain.AllStatKeys {
		if st, ok := byKey[k]; ok {
			out = append(out, st)
		}
	}
	return out, nil
}

func (s *Store) SaveStat(ctx context.Context, st *domain.Stat) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO stats (character_id, stat_key, value, level) VALUES ($1, $2, $3, $4)
		 ON CONFLICT (character_id, stat_key) DO UPDATE SET value = EXCLUDED.value, level = EXCLUDED.level`,
		st.CharacterID, string(st.Key), st.Value, st.Level)
	return err
}

// --- Activities & check-ins ---

func (s *Store) ListActivities(ctx context.Context) ([]domain.Activity, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT key, title, stat_key, base_xp, base_gold, has_duration, COALESCE(ref_minutes,0), rarity, COALESCE(icon,''), daily_cap
		   FROM activities ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Activity
	for rows.Next() {
		a, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

func (s *Store) GetActivity(ctx context.Context, key string) (*domain.Activity, error) {
	a, err := scanActivity(s.db.QueryRowContext(ctx,
		`SELECT key, title, stat_key, base_xp, base_gold, has_duration, COALESCE(ref_minutes,0), rarity, COALESCE(icon,''), daily_cap
		   FROM activities WHERE key = $1`, key))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	return a, err
}

func scanActivity(row rowScanner) (*domain.Activity, error) {
	var a domain.Activity
	var statKey string
	if err := row.Scan(&a.Key, &a.Title, &statKey, &a.BaseXP, &a.BaseGold,
		&a.HasDuration, &a.RefMinutes, &a.Rarity, &a.Icon, &a.DailyCap); err != nil {
		return nil, err
	}
	a.StatKey = domain.StatKey(statKey)
	return &a, nil
}

func (s *Store) TodayCheckins(ctx context.Context, characterID int64, localDate time.Time) ([]string, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT activity_key FROM activity_logs WHERE character_id = $1 AND local_date = $2`,
		characterID, localDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var k string
		if err := rows.Scan(&k); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) InsertActivityLog(ctx context.Context, log *domain.ActivityLog) error {
	var durationMin any
	if log.DurationMin != 0 {
		durationMin = log.DurationMin
	}
	var note any
	if log.Note != "" {
		note = log.Note
	}
	return s.db.QueryRowContext(ctx,
		`INSERT INTO activity_logs
		   (character_id, activity_key, duration_min, note, xp_awarded, gold_awarded, stat_awarded, is_crit, local_date)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id, created_at`,
		log.CharacterID, log.ActivityKey, durationMin, note,
		log.XPAwarded, log.GoldAwarded, log.StatAwarded, log.IsCrit, log.LocalDate,
	).Scan(&log.ID, &log.CreatedAt)
}

// --- Quests ---

func (s *Store) ListQuestsWithProgress(ctx context.Context, characterID int64) ([]domain.QuestWithProgress, error) {
	// Left-join active quests with the latest progress row for this character.
	rows, err := s.db.QueryContext(ctx,
		`SELECT q.id, q.title, q.type, q.condition, q.reward,
		        COALESCE(qp.progress, 0), COALESCE(qp.status, 'active')
		   FROM quests q
		   LEFT JOIN LATERAL (
		         SELECT progress, status FROM quest_progress p
		          WHERE p.quest_id = q.id AND p.character_id = $1
		          ORDER BY p.id DESC LIMIT 1
		   ) qp ON true
		  WHERE q.active = true
		  ORDER BY q.id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.QuestWithProgress
	for rows.Next() {
		var (
			qwp       domain.QuestWithProgress
			condition []byte
			reward    []byte
		)
		if err := rows.Scan(&qwp.ID, &qwp.Title, &qwp.Type, &condition, &reward, &qwp.Progress, &qwp.Status); err != nil {
			return nil, err
		}
		var cond map[string]any
		if err := unmarshalJSON(condition, &cond); err != nil {
			return nil, err
		}
		qwp.Target = questTarget(cond)
		qwp.Condition = cond
		if err := unmarshalJSON(reward, &qwp.Reward); err != nil {
			return nil, err
		}
		out = append(out, qwp)
	}
	return out, rows.Err()
}

// questTarget extracts the target value from a quest condition (best-effort).
func questTarget(cond map[string]any) int {
	for _, key := range []string{"target", "minutes", "streak_days", "count"} {
		if v, ok := cond[key]; ok {
			switch n := v.(type) {
			case float64:
				return int(n)
			case int:
				return n
			case int64:
				return int(n)
			}
		}
	}
	return 0
}

func (s *Store) UpsertQuestProgress(ctx context.Context, qp *domain.QuestProgress) error {
	var completedAt any
	if qp.CompletedAt != nil {
		completedAt = *qp.CompletedAt
	}
	var periodKey any
	if qp.PeriodKey != "" {
		periodKey = qp.PeriodKey
	}
	return s.db.QueryRowContext(ctx,
		`INSERT INTO quest_progress (character_id, quest_id, progress, status, period_key, completed_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (character_id, quest_id, period_key)
		 DO UPDATE SET progress = EXCLUDED.progress, status = EXCLUDED.status, completed_at = EXCLUDED.completed_at
		 RETURNING id`,
		qp.CharacterID, qp.QuestID, qp.Progress, qp.Status, periodKey, completedAt,
	).Scan(&qp.ID)
}

func (s *Store) ClaimQuest(ctx context.Context, characterID int64, questID string) (*domain.QuestReward, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Mark the latest completed (unclaimed) progress row as claimed.
	res, err := tx.ExecContext(ctx,
		`UPDATE quest_progress SET status = 'claimed'
		  WHERE id = (
		        SELECT id FROM quest_progress
		         WHERE character_id = $1 AND quest_id = $2 AND status = 'completed'
		         ORDER BY id DESC LIMIT 1
		  )`, characterID, questID)
	if err != nil {
		return nil, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, store.ErrNotFound
	}

	var reward []byte
	if err := tx.QueryRowContext(ctx, `SELECT reward FROM quests WHERE id = $1`, questID).Scan(&reward); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	var r domain.QuestReward
	if err := unmarshalJSON(reward, &r); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &r, nil
}

// --- Achievements ---

func (s *Store) ListAchievements(ctx context.Context, characterID int64) ([]domain.AchievementWithState, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT a.id, a.title, COALESCE(a.description,''), a.category, a.condition, a.reward, COALESCE(a.icon,''),
		        au.unlocked_at
		   FROM achievements a
		   LEFT JOIN achievement_unlocks au
		          ON au.achievement_id = a.id AND au.character_id = $1
		  ORDER BY a.id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.AchievementWithState
	for rows.Next() {
		var (
			aws       domain.AchievementWithState
			condition []byte
			reward    []byte
			unlocked  sql.NullTime
		)
		if err := rows.Scan(&aws.ID, &aws.Title, &aws.Description, &aws.Category,
			&condition, &reward, &aws.Icon, &unlocked); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(condition, &aws.Condition); err != nil {
			return nil, err
		}
		if err := unmarshalJSON(reward, &aws.Reward); err != nil {
			return nil, err
		}
		if unlocked.Valid {
			aws.Unlocked = true
			t := unlocked.Time
			aws.UnlockedAt = &t
		}
		out = append(out, aws)
	}
	return out, rows.Err()
}

func (s *Store) UnlockAchievement(ctx context.Context, characterID int64, achievementID string) error {
	// Idempotent: ON CONFLICT DO NOTHING on the (character_id, achievement_id) PK.
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO achievement_unlocks (character_id, achievement_id) VALUES ($1, $2)
		 ON CONFLICT (character_id, achievement_id) DO NOTHING`,
		characterID, achievementID)
	return err
}

// --- Shop & inventory ---

func (s *Store) ListShopItems(ctx context.Context) ([]domain.ShopItem, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, slot, rarity, price, effect, purchasable, COALESCE(icon,'')
		   FROM shop_items ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ShopItem
	for rows.Next() {
		it, err := scanShopItem(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *it)
	}
	return out, rows.Err()
}

func scanShopItem(row rowScanner) (*domain.ShopItem, error) {
	var (
		it     domain.ShopItem
		price  sql.NullInt64
		effect []byte
	)
	if err := row.Scan(&it.ID, &it.Name, &it.Slot, &it.Rarity, &price, &effect, &it.Purchasable, &it.Icon); err != nil {
		return nil, err
	}
	if price.Valid {
		p := int(price.Int64)
		it.Price = &p
	}
	if err := unmarshalJSON(effect, &it.Effect); err != nil {
		return nil, err
	}
	return &it, nil
}

func (s *Store) BuyItem(ctx context.Context, characterID int64, itemID string) (newGold int64, inventoryItemID int64, err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	// Lock the character row and read its gold balance.
	var gold int64
	if err := tx.QueryRowContext(ctx,
		`SELECT gold FROM characters WHERE id = $1 FOR UPDATE`, characterID).Scan(&gold); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, store.ErrNotFound
		}
		return 0, 0, err
	}

	// Read the item price; must be purchasable and for sale.
	var price sql.NullInt64
	var purchasable bool
	if err := tx.QueryRowContext(ctx,
		`SELECT price, purchasable FROM shop_items WHERE id = $1`, itemID).Scan(&price, &purchasable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, store.ErrNotFound
		}
		return 0, 0, err
	}
	if !purchasable || !price.Valid {
		return 0, 0, store.ErrNotFound
	}
	if gold < price.Int64 {
		return 0, 0, store.ErrInsufficientGold
	}

	newGold = gold - price.Int64
	if _, err := tx.ExecContext(ctx, `UPDATE characters SET gold = $2 WHERE id = $1`, characterID, newGold); err != nil {
		return 0, 0, err
	}

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO inventory_items (character_id, shop_item_id, acquired_via, quantity)
		 VALUES ($1, $2, 'purchase', 1) RETURNING id`,
		characterID, itemID).Scan(&inventoryItemID); err != nil {
		return 0, 0, err
	}

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO transactions (character_id, amount, reason, ref_id)
		 VALUES ($1, $2, 'purchase', $3)`,
		characterID, -price.Int64, itemID); err != nil {
		return 0, 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, err
	}
	return newGold, inventoryItemID, nil
}

func (s *Store) ListInventory(ctx context.Context, characterID int64) ([]domain.InventoryItem, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, character_id, shop_item_id, acquired_via, quantity, acquired_at
		   FROM inventory_items WHERE character_id = $1 ORDER BY id`, characterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.InventoryItem
	for rows.Next() {
		var it domain.InventoryItem
		if err := rows.Scan(&it.ID, &it.CharacterID, &it.ShopItemID, &it.AcquiredVia, &it.Quantity, &it.AcquiredAt); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (s *Store) EquipItem(ctx context.Context, characterID int64, inventoryItemID int64) (map[string]int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Resolve the slot of the inventory item via its shop_item, ensuring it
	// belongs to this character.
	var slot string
	if err := tx.QueryRowContext(ctx,
		`SELECT si.slot
		   FROM inventory_items ii
		   JOIN shop_items si ON si.id = ii.shop_item_id
		  WHERE ii.id = $1 AND ii.character_id = $2`,
		inventoryItemID, characterID).Scan(&slot); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}

	equipped, err := lockEquippedTx(ctx, tx, characterID)
	if err != nil {
		return nil, err
	}
	equipped[slot] = inventoryItemID
	if err := saveEquippedTx(ctx, tx, characterID, equipped); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return equipped, nil
}

func (s *Store) UnequipItem(ctx context.Context, characterID int64, inventoryItemID int64) (map[string]int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	equipped, err := lockEquippedTx(ctx, tx, characterID)
	if err != nil {
		return nil, err
	}
	for slot, id := range equipped {
		if id == inventoryItemID {
			delete(equipped, slot)
			break
		}
	}
	if err := saveEquippedTx(ctx, tx, characterID, equipped); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return equipped, nil
}

// lockEquippedTx selects and row-locks the character's equipped map.
func lockEquippedTx(ctx context.Context, tx *sql.Tx, characterID int64) (map[string]int64, error) {
	var raw []byte
	if err := tx.QueryRowContext(ctx,
		`SELECT equipped FROM characters WHERE id = $1 FOR UPDATE`, characterID).Scan(&raw); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, store.ErrNotFound
		}
		return nil, err
	}
	equipped := map[string]int64{}
	if err := unmarshalJSON(raw, &equipped); err != nil {
		return nil, err
	}
	return equipped, nil
}

// saveEquippedTx writes the equipped map back to the character row.
func saveEquippedTx(ctx context.Context, tx *sql.Tx, characterID int64, equipped map[string]int64) error {
	b, err := marshalJSON(equipped)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE characters SET equipped = $2 WHERE id = $1`, characterID, b)
	return err
}

// --- Transactions ---

func (s *Store) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	var refID any
	if tx.RefID != "" {
		refID = tx.RefID
	}
	return s.db.QueryRowContext(ctx,
		`INSERT INTO transactions (character_id, amount, reason, ref_id)
		 VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		tx.CharacterID, tx.Amount, tx.Reason, refID,
	).Scan(&tx.ID, &tx.CreatedAt)
}

// --- Reports & notifications ---

func (s *Store) GetReportToday(ctx context.Context, characterID int64, localDate time.Time) (*domain.DailyReportView, error) {
	ch, err := scanCharacter(s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, level, xp_total, gold, class, rank, streak_days, best_streak, last_checkin_date, equipped
		   FROM characters WHERE id = $1`, characterID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, store.ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	// Aggregate today's logs per activity (joined with the catalog for title/stat).
	rows, err := s.db.QueryContext(ctx,
		`SELECT l.activity_key, COALESCE(a.title, l.activity_key), COALESCE(a.stat_key, ''),
		        SUM(l.xp_awarded)::int, SUM(l.gold_awarded)::int, COUNT(*)::int, bool_or(l.is_crit)
		   FROM activity_logs l
		   LEFT JOIN activities a ON a.key = l.activity_key
		  WHERE l.character_id = $1 AND l.local_date = $2
		  GROUP BY l.activity_key, a.title, a.stat_key
		  ORDER BY l.activity_key`, characterID, localDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	view := &domain.DailyReportView{
		Date:        localDate,
		Entries:     []domain.ReportEntry{},
		StreakDays:  ch.StreakDays,
		StreakMult:  1.0,
		Level:       ch.Level,
		XPIntoLevel: ch.XPTotal,
	}
	for rows.Next() {
		var (
			e       domain.ReportEntry
			statKey string
			gold    int
		)
		if err := rows.Scan(&e.ActivityKey, &e.Title, &statKey, &e.XP, &gold, &e.Count, &e.IsCrit); err != nil {
			return nil, err
		}
		e.StatKey = domain.StatKey(statKey)
		view.Entries = append(view.Entries, e)
		view.TotalXP += e.XP
		view.TotalGold += gold
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	view.HadActivity = len(view.Entries) > 0

	// Open quests (not claimed/expired).
	open, err := s.openQuests(ctx, characterID)
	if err != nil {
		return nil, err
	}
	view.OpenQuests = open
	return view, nil
}

// openQuests returns active quests whose progress is not claimed/expired.
func (s *Store) openQuests(ctx context.Context, characterID int64) ([]domain.QuestWithProgress, error) {
	all, err := s.ListQuestsWithProgress(ctx, characterID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.QuestWithProgress, 0, len(all))
	for _, q := range all {
		if q.Status == "claimed" || q.Status == "expired" {
			continue
		}
		out = append(out, q)
	}
	return out, nil
}

func (s *Store) MarkReportSent(ctx context.Context, userID int64, reportDate time.Time, kind string) (bool, error) {
	// The UNIQUE (user_id, report_date, kind) index gives idempotency: if the
	// row already exists, ON CONFLICT DO NOTHING affects 0 rows.
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO daily_reports (user_id, report_date, kind) VALUES ($1, $2, $3)
		 ON CONFLICT (user_id, report_date, kind) DO NOTHING`,
		userID, reportDate, kind)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Store) UsersForNotificationSlot(ctx context.Context, now time.Time, slotHour int, kind string) ([]domain.User, error) {
	// Select users whose local hour (now in their timezone) matches slotHour and
	// who have no daily_reports row of this kind for their local date.
	//
	// timezone() converts the UTC instant into the user's local wall-clock time;
	// EXTRACT(hour ...) and ::date then give the local hour and local date used
	// for the slot match and the idempotency check.
	rows, err := s.db.QueryContext(ctx,
		`SELECT u.id, u.telegram_user_id, u.device_id, u.username, u.timezone, u.notif_prefs, u.created_at, u.last_seen_at
		   FROM users u
		  WHERE EXTRACT(hour FROM (timezone(u.timezone, $1))) = $2
		    AND NOT EXISTS (
		        SELECT 1 FROM daily_reports d
		         WHERE d.user_id = u.id
		           AND d.kind = $3
		           AND d.report_date = (timezone(u.timezone, $1))::date
		    )
		  ORDER BY u.id`, now.UTC(), slotHour, kind)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		if !notifEnabled(u.NotifPrefs, kind) {
			continue
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// notifEnabled reports whether the user opted in to the given notification kind.
func notifEnabled(p domain.NotifPrefs, kind string) bool {
	switch kind {
	case "daily":
		return p.Daily
	case "streak_reminder":
		return p.StreakReminder
	case "morning":
		return p.Morning
	case "milestone":
		return p.Milestone
	default:
		return true
	}
}

func (s *Store) UpdateNotificationSettings(ctx context.Context, userID int64, tz string, prefs domain.NotifPrefs) error {
	b, err := marshalJSON(prefs)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET timezone = COALESCE(NULLIF($2,''), timezone), notif_prefs = $3 WHERE id = $1`,
		userID, tz, b)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return store.ErrNotFound
	}
	return nil
}
