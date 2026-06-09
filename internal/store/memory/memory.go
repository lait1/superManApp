// Package memory provides an in-memory store.Store implementation. It is the
// default store when DATABASE_URL is unset (docs/07 §5) and is the primary
// store for the offline demo: it is fully functional and thread-safe.
//
// All mutating and reading methods take the RWMutex; the catalog tables
// (activities, quests, achievements, shop_items) are seeded once in New (see
// seed.go) and are then read-only.
package memory

import (
	"context"
	"sort"
	"sync"
	"time"

	"superMen/internal/domain"
	"superMen/internal/store"
)

// Store is an in-memory implementation of store.Store, suitable for dev, the
// offline demo and tests. The mutex guards every map below.
type Store struct {
	// StartingGold is granted to newly created characters (dev/shop testing).
	// Set it right after New(), before serving requests.
	StartingGold int64

	mu sync.RWMutex

	// id sequences (BIGSERIAL emulation)
	nextUserID   int64
	nextCharID   int64
	nextLogID    int64
	nextQPID     int64
	nextInvID    int64
	nextTxID     int64
	nextReportID int64

	// user data, keyed by primary id
	users      map[int64]*domain.User
	characters map[int64]*domain.Character               // characterID -> character
	charByUser map[int64]int64                           // userID -> characterID
	stats      map[int64]map[domain.StatKey]*domain.Stat // characterID -> key -> stat

	// indexes for identity lookup
	userByTelegram map[int64]int64  // telegramUserID -> userID
	userByDevice   map[string]int64 // deviceID -> userID

	// per-character collections
	logs          map[int64][]*domain.ActivityLog                // characterID -> logs
	questProgress map[int64]map[string]*domain.QuestProgress     // characterID -> questID(+periodKey) -> progress
	unlocks       map[int64]map[string]*domain.AchievementUnlock // characterID -> achievementID -> unlock
	inventory     map[int64][]*domain.InventoryItem              // characterID -> items
	transactions  map[int64][]*domain.Transaction                // characterID -> ledger

	// notification idempotency: userID -> "YYYY-MM-DD|kind" -> sent
	reports map[int64]map[string]*domain.DailyReport

	// read-only catalogs (seeded in New)
	activities    []domain.Activity
	activityByKey map[string]domain.Activity
	quests        []domain.Quest
	questByID     map[string]domain.Quest
	achievements  []domain.Achievement
	achByID       map[string]domain.Achievement
	shopItems     []domain.ShopItem
	shopByID      map[string]domain.ShopItem
}

// New constructs an in-memory Store with the catalogs seeded.
func New() *Store {
	s := &Store{
		nextUserID:   1,
		nextCharID:   1,
		nextLogID:    1,
		nextQPID:     1,
		nextInvID:    1,
		nextTxID:     1,
		nextReportID: 1,

		users:      make(map[int64]*domain.User),
		characters: make(map[int64]*domain.Character),
		charByUser: make(map[int64]int64),
		stats:      make(map[int64]map[domain.StatKey]*domain.Stat),

		userByTelegram: make(map[int64]int64),
		userByDevice:   make(map[string]int64),

		logs:          make(map[int64][]*domain.ActivityLog),
		questProgress: make(map[int64]map[string]*domain.QuestProgress),
		unlocks:       make(map[int64]map[string]*domain.AchievementUnlock),
		inventory:     make(map[int64][]*domain.InventoryItem),
		transactions:  make(map[int64][]*domain.Transaction),

		reports: make(map[int64]map[string]*domain.DailyReport),
	}
	s.seedCatalogs()
	return s
}

// compile-time assertion that *Store satisfies store.Store.
var _ store.Store = (*Store)(nil)

// --- helpers (callers hold the appropriate lock) ---

// dayKey formats a date as the local YYYY-MM-DD key used for daily caps,
// reports and notification idempotency.
func dayKey(t time.Time) string {
	return t.Format("2006-01-02")
}

// sameLocalDate reports whether a and b fall on the same calendar day.
func sameLocalDate(a, b time.Time) bool {
	return dayKey(a) == dayKey(b)
}

// createCharacterLocked initializes a character and its 5 stat rows for a user.
// The caller must hold the write lock and must have already created the user.
func (s *Store) createCharacterLocked(userID int64) *domain.Character {
	id := s.nextCharID
	s.nextCharID++
	ch := &domain.Character{
		ID:         id,
		UserID:     userID,
		Name:       "superMen",
		Level:      1,
		XPTotal:    0,
		Gold:       s.StartingGold,
		Class:      "adventurer",
		Rank:       "recruit",
		StreakDays: 0,
		BestStreak: 0,
		Equipped:   map[string]int64{},
		Appearance: domain.DefaultAppearance(),
		Onboarded:  false,
	}
	s.characters[id] = ch
	s.charByUser[userID] = id

	stat := make(map[domain.StatKey]*domain.Stat, len(domain.AllStatKeys))
	for _, k := range domain.AllStatKeys {
		stat[k] = &domain.Stat{CharacterID: id, Key: k, Value: 0, Level: 1}
	}
	s.stats[id] = stat
	return ch
}

// cloneCharacter returns a deep copy so callers cannot mutate stored state.
func cloneCharacter(ch *domain.Character) *domain.Character {
	if ch == nil {
		return nil
	}
	cp := *ch
	cp.Equipped = make(map[string]int64, len(ch.Equipped))
	for k, v := range ch.Equipped {
		cp.Equipped[k] = v
	}
	return &cp
}

// cloneUser returns a deep copy of a user.
func cloneUser(u *domain.User) domain.User {
	cp := *u
	if u.TelegramUserID != nil {
		v := *u.TelegramUserID
		cp.TelegramUserID = &v
	}
	if u.DeviceID != nil {
		v := *u.DeviceID
		cp.DeviceID = &v
	}
	if u.LastSeenAt != nil {
		v := *u.LastSeenAt
		cp.LastSeenAt = &v
	}
	return cp
}

// --- Users & identity ---

func (s *Store) GetOrCreateUserByTelegramID(ctx context.Context, telegramUserID int64, username string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if uid, ok := s.userByTelegram[telegramUserID]; ok {
		u := s.users[uid]
		now := time.Now().UTC()
		u.LastSeenAt = &now
		if username != "" {
			u.Username = username
		}
		cp := cloneUser(u)
		return &cp, nil
	}

	id := s.nextUserID
	s.nextUserID++
	tid := telegramUserID
	now := time.Now().UTC()
	u := &domain.User{
		ID:             id,
		TelegramUserID: &tid,
		Username:       username,
		Timezone:       "UTC",
		NotifPrefs:     defaultNotifPrefs(),
		CreatedAt:      now,
		LastSeenAt:     &now,
	}
	s.users[id] = u
	s.userByTelegram[telegramUserID] = id
	s.createCharacterLocked(id)

	cp := cloneUser(u)
	return &cp, nil
}

func (s *Store) GetOrCreateUserByDeviceID(ctx context.Context, deviceID string) (*domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if uid, ok := s.userByDevice[deviceID]; ok {
		u := s.users[uid]
		now := time.Now().UTC()
		u.LastSeenAt = &now
		cp := cloneUser(u)
		return &cp, nil
	}

	id := s.nextUserID
	s.nextUserID++
	dev := deviceID
	now := time.Now().UTC()
	u := &domain.User{
		ID:         id,
		DeviceID:   &dev,
		Username:   "guest",
		Timezone:   "UTC",
		NotifPrefs: defaultNotifPrefs(),
		CreatedAt:  now,
		LastSeenAt: &now,
	}
	s.users[id] = u
	s.userByDevice[deviceID] = id
	s.createCharacterLocked(id)

	cp := cloneUser(u)
	return &cp, nil
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

// --- Character & stats ---

func (s *Store) GetCharacter(ctx context.Context, userID int64) (*domain.Character, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cid, ok := s.charByUser[userID]
	if !ok {
		return nil, store.ErrNotFound
	}
	return cloneCharacter(s.characters[cid]), nil
}

func (s *Store) SaveCharacter(ctx context.Context, ch *domain.Character) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cur, ok := s.characters[ch.ID]
	if !ok {
		return store.ErrNotFound
	}
	// Persist denormalized fields in place, preserving identity.
	cur.Name = ch.Name
	cur.Level = ch.Level
	cur.XPTotal = ch.XPTotal
	cur.Gold = ch.Gold
	cur.Class = ch.Class
	cur.Rank = ch.Rank
	cur.StreakDays = ch.StreakDays
	cur.BestStreak = ch.BestStreak
	cur.Appearance = ch.Appearance
	cur.Onboarded = ch.Onboarded
	if ch.LastCheckinDate != nil {
		d := *ch.LastCheckinDate
		cur.LastCheckinDate = &d
	} else {
		cur.LastCheckinDate = nil
	}
	cur.Equipped = make(map[string]int64, len(ch.Equipped))
	for k, v := range ch.Equipped {
		cur.Equipped[k] = v
	}
	return nil
}

func (s *Store) GetStats(ctx context.Context, characterID int64) ([]domain.Stat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.stats[characterID]
	if !ok {
		return nil, store.ErrNotFound
	}
	out := make([]domain.Stat, 0, len(domain.AllStatKeys))
	for _, k := range domain.AllStatKeys {
		if st := m[k]; st != nil {
			out = append(out, *st)
		}
	}
	return out, nil
}

func (s *Store) SaveStat(ctx context.Context, st *domain.Stat) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.stats[st.CharacterID]
	if !ok {
		return store.ErrNotFound
	}
	cp := *st
	m[st.Key] = &cp
	return nil
}

// --- Activities & check-ins ---

func (s *Store) ListActivities(ctx context.Context) ([]domain.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Activity, len(s.activities))
	copy(out, s.activities)
	return out, nil
}

func (s *Store) GetActivity(ctx context.Context, key string) (*domain.Activity, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	a, ok := s.activityByKey[key]
	if !ok {
		return nil, store.ErrNotFound
	}
	cp := a
	return &cp, nil
}

func (s *Store) TodayCheckins(ctx context.Context, characterID int64, localDate time.Time) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []string
	for _, lg := range s.logs[characterID] {
		if sameLocalDate(lg.LocalDate, localDate) {
			out = append(out, lg.ActivityKey)
		}
	}
	return out, nil
}

func (s *Store) InsertActivityLog(ctx context.Context, log *domain.ActivityLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *log
	if cp.ID == 0 {
		cp.ID = s.nextLogID
		s.nextLogID++
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now().UTC()
	}
	s.logs[log.CharacterID] = append(s.logs[log.CharacterID], &cp)
	log.ID = cp.ID
	return nil
}

// --- Quests ---

// qpKey builds the per-character quest_progress map key, combining the quest id
// with its period (daily/weekly) so distinct periods do not collide.
func qpKey(questID, periodKey string) string {
	if periodKey == "" {
		return questID
	}
	return questID + "|" + periodKey
}

func (s *Store) ListQuestsWithProgress(ctx context.Context, characterID int64) ([]domain.QuestWithProgress, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prog := s.questProgress[characterID]
	out := make([]domain.QuestWithProgress, 0, len(s.quests))
	for _, q := range s.quests {
		if !q.Active {
			continue
		}
		qwp := domain.QuestWithProgress{
			ID:        q.ID,
			Title:     q.Title,
			Type:      q.Type,
			Target:    questTarget(q),
			Status:    "active",
			Reward:    q.Reward,
			Condition: q.Condition,
		}
		// Find the most recent progress row for this quest (any period).
		if prog != nil {
			var best *domain.QuestProgress
			for _, p := range prog {
				if p.QuestID != q.ID {
					continue
				}
				if best == nil || p.ID > best.ID {
					best = p
				}
			}
			if best != nil {
				qwp.Progress = best.Progress
				qwp.Status = best.Status
				if best.Target > 0 {
					qwp.Target = best.Target
				}
			}
		}
		out = append(out, qwp)
	}
	// Deterministic order by id.
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// questTarget extracts the target value from a quest's condition (best-effort).
func questTarget(q domain.Quest) int {
	for _, key := range []string{"target", "minutes", "streak_days", "count"} {
		if v, ok := q.Condition[key]; ok {
			switch n := v.(type) {
			case int:
				return n
			case int64:
				return int(n)
			case float64:
				return int(n)
			}
		}
	}
	return 0
}

func (s *Store) UpsertQuestProgress(ctx context.Context, qp *domain.QuestProgress) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.questProgress[qp.CharacterID]
	if !ok {
		m = make(map[string]*domain.QuestProgress)
		s.questProgress[qp.CharacterID] = m
	}
	key := qpKey(qp.QuestID, qp.PeriodKey)
	if cur, found := m[key]; found {
		cur.Progress = qp.Progress
		cur.Target = qp.Target
		cur.Status = qp.Status
		if qp.CompletedAt != nil {
			t := *qp.CompletedAt
			cur.CompletedAt = &t
		}
		qp.ID = cur.ID
		return nil
	}
	cp := *qp
	cp.ID = s.nextQPID
	s.nextQPID++
	if cp.CompletedAt != nil {
		t := *qp.CompletedAt
		cp.CompletedAt = &t
	}
	m[key] = &cp
	qp.ID = cp.ID
	return nil
}

func (s *Store) ClaimQuest(ctx context.Context, characterID int64, questID string) (*domain.QuestReward, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.questByID[questID]
	if !ok {
		return nil, store.ErrNotFound
	}
	m := s.questProgress[characterID]
	if m == nil {
		return nil, store.ErrNotFound
	}
	// Find a completed (not yet claimed) progress row for this quest.
	var target *domain.QuestProgress
	for _, p := range m {
		if p.QuestID == questID && p.Status == "completed" {
			if target == nil || p.ID > target.ID {
				target = p
			}
		}
	}
	if target == nil {
		return nil, store.ErrNotFound
	}
	target.Status = "claimed"
	reward := q.Reward
	return &reward, nil
}

// --- Achievements ---

func (s *Store) ListAchievements(ctx context.Context, characterID int64) ([]domain.AchievementWithState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	unlocked := s.unlocks[characterID]
	out := make([]domain.AchievementWithState, 0, len(s.achievements))
	for _, a := range s.achievements {
		aws := domain.AchievementWithState{Achievement: a}
		if unlocked != nil {
			if u, ok := unlocked[a.ID]; ok {
				aws.Unlocked = true
				t := u.UnlockedAt
				aws.UnlockedAt = &t
			}
		}
		out = append(out, aws)
	}
	return out, nil
}

func (s *Store) UnlockAchievement(ctx context.Context, characterID int64, achievementID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.achByID[achievementID]; !ok {
		return store.ErrNotFound
	}
	m, ok := s.unlocks[characterID]
	if !ok {
		m = make(map[string]*domain.AchievementUnlock)
		s.unlocks[characterID] = m
	}
	if _, exists := m[achievementID]; exists {
		return nil // idempotent
	}
	m[achievementID] = &domain.AchievementUnlock{
		CharacterID:   characterID,
		AchievementID: achievementID,
		UnlockedAt:    time.Now().UTC(),
	}
	return nil
}

// --- Shop & inventory ---

func (s *Store) ListShopItems(ctx context.Context) ([]domain.ShopItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.ShopItem, len(s.shopItems))
	copy(out, s.shopItems)
	return out, nil
}

func (s *Store) BuyItem(ctx context.Context, characterID int64, itemID string) (newGold int64, inventoryItemID int64, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.characters[characterID]
	if !ok {
		return 0, 0, store.ErrNotFound
	}
	item, ok := s.shopByID[itemID]
	if !ok {
		return 0, 0, store.ErrNotFound
	}
	if !item.Purchasable || item.Price == nil {
		return 0, 0, store.ErrNotFound
	}
	price := int64(*item.Price)
	if ch.Gold < price {
		return 0, 0, store.ErrInsufficientGold
	}

	// Transactional: debit gold, append ledger row, add inventory item.
	ch.Gold -= price

	invID := s.nextInvID
	s.nextInvID++
	inv := &domain.InventoryItem{
		ID:          invID,
		CharacterID: characterID,
		ShopItemID:  itemID,
		AcquiredVia: "purchase",
		Quantity:    1,
		AcquiredAt:  time.Now().UTC(),
	}
	s.inventory[characterID] = append(s.inventory[characterID], inv)

	txID := s.nextTxID
	s.nextTxID++
	s.transactions[characterID] = append(s.transactions[characterID], &domain.Transaction{
		ID:          txID,
		CharacterID: characterID,
		Amount:      -*item.Price,
		Reason:      "purchase",
		RefID:       itemID,
		CreatedAt:   time.Now().UTC(),
	})

	return ch.Gold, invID, nil
}

func (s *Store) ListInventory(ctx context.Context, characterID int64) ([]domain.InventoryItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := s.inventory[characterID]
	out := make([]domain.InventoryItem, 0, len(items))
	for _, it := range items {
		out = append(out, *it)
	}
	return out, nil
}

func (s *Store) EquipItem(ctx context.Context, characterID int64, inventoryItemID int64) (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.characters[characterID]
	if !ok {
		return nil, store.ErrNotFound
	}
	var inv *domain.InventoryItem
	for _, it := range s.inventory[characterID] {
		if it.ID == inventoryItemID {
			inv = it
			break
		}
	}
	if inv == nil {
		return nil, store.ErrNotFound
	}
	item, ok := s.shopByID[inv.ShopItemID]
	if !ok {
		return nil, store.ErrNotFound
	}
	if ch.Equipped == nil {
		ch.Equipped = map[string]int64{}
	}
	ch.Equipped[item.Slot] = inventoryItemID
	return cloneEquipped(ch.Equipped), nil
}

func (s *Store) UnequipItem(ctx context.Context, characterID int64, inventoryItemID int64) (map[string]int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ch, ok := s.characters[characterID]
	if !ok {
		return nil, store.ErrNotFound
	}
	if ch.Equipped == nil {
		ch.Equipped = map[string]int64{}
	}
	for slot, id := range ch.Equipped {
		if id == inventoryItemID {
			delete(ch.Equipped, slot)
			break
		}
	}
	return cloneEquipped(ch.Equipped), nil
}

func cloneEquipped(m map[string]int64) map[string]int64 {
	out := make(map[string]int64, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// --- Transactions ---

func (s *Store) AddTransaction(ctx context.Context, tx *domain.Transaction) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cp := *tx
	if cp.ID == 0 {
		cp.ID = s.nextTxID
		s.nextTxID++
	}
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now().UTC()
	}
	s.transactions[tx.CharacterID] = append(s.transactions[tx.CharacterID], &cp)
	tx.ID = cp.ID
	return nil
}

// --- Reports & notifications ---

func (s *Store) GetReportToday(ctx context.Context, characterID int64, localDate time.Time) (*domain.DailyReportView, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ch, ok := s.characters[characterID]
	if !ok {
		return nil, store.ErrNotFound
	}

	// Aggregate today's logs per activity key.
	type agg struct {
		entry domain.ReportEntry
	}
	order := make([]string, 0)
	byKey := make(map[string]*agg)
	totalXP := 0
	totalGold := 0
	for _, lg := range s.logs[characterID] {
		if !sameLocalDate(lg.LocalDate, localDate) {
			continue
		}
		totalXP += lg.XPAwarded
		totalGold += lg.GoldAwarded
		a, exists := byKey[lg.ActivityKey]
		if !exists {
			title := lg.ActivityKey
			var sk domain.StatKey
			if cat, ok := s.activityByKey[lg.ActivityKey]; ok {
				title = cat.Title
				sk = cat.StatKey
			}
			a = &agg{entry: domain.ReportEntry{
				ActivityKey: lg.ActivityKey,
				Title:       title,
				StatKey:     sk,
			}}
			byKey[lg.ActivityKey] = a
			order = append(order, lg.ActivityKey)
		}
		a.entry.XP += lg.XPAwarded
		a.entry.Count++
		if lg.IsCrit {
			a.entry.IsCrit = true
		}
	}

	entries := make([]domain.ReportEntry, 0, len(order))
	for _, k := range order {
		entries = append(entries, byKey[k].entry)
	}

	// Open (active/completed but unclaimed) quests for the report.
	openQuests := make([]domain.QuestWithProgress, 0)
	prog := s.questProgress[characterID]
	for _, q := range s.quests {
		if !q.Active {
			continue
		}
		status := "active"
		progress := 0
		target := questTarget(q)
		if prog != nil {
			var best *domain.QuestProgress
			for _, p := range prog {
				if p.QuestID == q.ID && (best == nil || p.ID > best.ID) {
					best = p
				}
			}
			if best != nil {
				status = best.Status
				progress = best.Progress
				if best.Target > 0 {
					target = best.Target
				}
			}
		}
		if status == "claimed" || status == "expired" {
			continue
		}
		openQuests = append(openQuests, domain.QuestWithProgress{
			ID:       q.ID,
			Title:    q.Title,
			Type:     q.Type,
			Progress: progress,
			Target:   target,
			Status:   status,
			Reward:   q.Reward,
		})
	}
	sort.Slice(openQuests, func(i, j int) bool { return openQuests[i].ID < openQuests[j].ID })

	view := &domain.DailyReportView{
		Date:        localDate,
		Entries:     entries,
		TotalXP:     totalXP,
		TotalGold:   totalGold,
		StreakDays:  ch.StreakDays,
		StreakMult:  1.0,
		Level:       ch.Level,
		XPIntoLevel: ch.XPTotal,
		XPToNext:    0,
		OpenQuests:  openQuests,
		HadActivity: len(entries) > 0,
	}
	return view, nil
}

func (s *Store) MarkReportSent(ctx context.Context, userID int64, reportDate time.Time, kind string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := dayKey(reportDate) + "|" + kind
	m, ok := s.reports[userID]
	if !ok {
		m = make(map[string]*domain.DailyReport)
		s.reports[userID] = m
	}
	if _, exists := m[key]; exists {
		return false, nil // already sent
	}
	id := s.nextReportID
	s.nextReportID++
	m[key] = &domain.DailyReport{
		ID:         id,
		UserID:     userID,
		ReportDate: reportDate,
		Kind:       kind,
		SentAt:     time.Now().UTC(),
	}
	return true, nil
}

func (s *Store) UsersForNotificationSlot(ctx context.Context, now time.Time, slotHour int, kind string) ([]domain.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var out []domain.User
	for _, u := range s.users {
		if !notifEnabled(u.NotifPrefs, kind) {
			continue
		}
		loc, err := time.LoadLocation(u.Timezone)
		if err != nil || loc == nil {
			loc = time.UTC
		}
		local := now.In(loc)
		if local.Hour() != slotHour {
			continue
		}
		// Skip if already notified of this kind today (in the user's tz).
		if m := s.reports[u.ID]; m != nil {
			if _, sent := m[dayKey(local)+"|"+kind]; sent {
				continue
			}
		}
		out = append(out, cloneUser(u))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
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
	s.mu.Lock()
	defer s.mu.Unlock()

	u, ok := s.users[userID]
	if !ok {
		return store.ErrNotFound
	}
	if tz != "" {
		u.Timezone = tz
	}
	u.NotifPrefs = prefs
	return nil
}
