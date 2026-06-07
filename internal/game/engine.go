// Package game holds the domain logic of superMen: reward computation, level
// and stat curves, streak multipliers and class/rank recalculation. Formulas
// are specified in docs/03-progression-and-stats.md.
package game

import (
	"context"
	"math"
	"math/rand"
	"sort"
	"time"

	"superMen/internal/config"
	"superMen/internal/domain"
	"superMen/internal/store"
)

// --- class & rank string constants (docs/01 §4, §6; values match the asset
// generator in cmd/genassets and the seed defaults in docs/08 §2). ---

// Character classes derived from the dominant stat.
const (
	ClassWarrior    = "warrior"    // STR
	ClassSage       = "sage"       // INT
	ClassPaladin    = "paladin"    // DIS
	ClassDruid      = "druid"      // VIT
	ClassBard       = "bard"       // CHA
	ClassAdventurer = "adventurer" // balanced
)

// adventurerSpread is the maximum (max-min) stat-level spread that still counts
// as "balanced" → Adventurer (docs/03 §5).
const adventurerSpread = 2

// Character ranks derived from the character level (docs/01 §6).
const (
	RankRecruit = "recruit" // 1–9
	RankSeeker  = "seeker"  // 10–24
	RankVeteran = "veteran" // 25–49
	RankMaster  = "master"  // 50–99
	RankLegend  = "legend"  // 100+
)

// classByStat maps the dominant stat key to its class name (docs/01 §4).
var classByStat = map[domain.StatKey]string{
	domain.StatSTR: ClassWarrior,
	domain.StatINT: ClassSage,
	domain.StatDIS: ClassPaladin,
	domain.StatVIT: ClassDruid,
	domain.StatCHA: ClassBard,
}

// Engine computes check-in rewards and progression using the configured
// balance constants and the persistence store.
type Engine struct {
	store   store.Store
	balance config.Balance

	// randFloat returns a pseudo-random float64 in [0,1); it is the only source
	// of non-determinism in the engine and is injected so that crit/drop rolls
	// can be mocked in tests (docs/03 §4). Defaults to a seeded math/rand source
	// in New; tests overwrite it directly (same package) or via SetRandSource.
	randFloat func() float64

	// now returns the current time; injected for deterministic local-date math
	// in tests. Defaults to time.Now.
	now func() time.Time
}

// New constructs an Engine backed by the given store and balance constants.
func New(st store.Store, balance config.Balance) *Engine {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return &Engine{
		store:     st,
		balance:   balance,
		randFloat: r.Float64,
		now:       time.Now,
	}
}

// SetRandSource overrides the engine's random source. Passing a function that
// returns a fixed value makes crit/drop rolls deterministic in tests; passing
// nil restores a default seeded source.
func (e *Engine) SetRandSource(f func() float64) {
	if f == nil {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		f = r.Float64
	}
	e.randFloat = f
}

// XPToNext returns the XP required to advance from the given character level to
// the next one: round(XP_BASE * level^XP_EXP) (docs/03 §2). Levels below 1 are
// treated as level 1.
func (e *Engine) XPToNext(level int) int64 {
	if level < 1 {
		level = 1
	}
	return int64(math.Round(e.balance.XPBase * math.Pow(float64(level), e.balance.XPExp)))
}

// StatToNext returns the points required to advance a stat from the given level
// to the next: round(STAT_BASE * level^STAT_EXP) (docs/03 §3). Levels below 1
// are treated as level 1.
func (e *Engine) StatToNext(level int) int64 {
	if level < 1 {
		level = 1
	}
	return int64(math.Round(e.balance.StatBase * math.Pow(float64(level), e.balance.StatExp)))
}

// xpTotalForLevel returns the cumulative XP required to *be* at the given level:
// Σ XPToNext(i) for i = 1..level-1 (docs/03 §2). Level 1 needs 0 XP.
func (e *Engine) xpTotalForLevel(level int) int64 {
	var sum int64
	for i := 1; i < level; i++ {
		sum += e.XPToNext(i)
	}
	return sum
}

// statTotalForLevel returns the cumulative points required to *be* at the given
// stat level: Σ StatToNext(i) for i = 1..level-1 (docs/03 §3).
func (e *Engine) statTotalForLevel(level int) int64 {
	var sum int64
	for i := 1; i < level; i++ {
		sum += e.StatToNext(i)
	}
	return sum
}

// LevelForXP returns the character level reached with the given total XP. The
// minimum level is 1.
func (e *Engine) LevelForXP(xpTotal int64) int {
	level := 1
	for xpTotal >= e.xpTotalForLevel(level+1) {
		level++
	}
	return level
}

// StatLevelForValue returns the stat level reached with the given accumulated
// stat value. The minimum level is 1.
func (e *Engine) StatLevelForValue(value int64) int {
	level := 1
	for value >= e.statTotalForLevel(level+1) {
		level++
	}
	return level
}

// StreakMult returns the XP multiplier for a streak of the given number of
// consecutive days, using the configured streak tiers: the largest tier whose
// lower-bound days ≤ the streak (docs/03 §6). Returns 1.0 when no tier matches.
func (e *Engine) StreakMult(days int) float64 {
	mult := 1.0
	best := -1
	for _, t := range e.balance.StreakTiers {
		if days >= t.Days && t.Days > best {
			best = t.Days
			mult = t.Mult
		}
	}
	return mult
}

// RankForLevel returns the rank band for a character level (docs/01 §6).
func RankForLevel(level int) string {
	switch {
	case level >= 100:
		return RankLegend
	case level >= 50:
		return RankMaster
	case level >= 25:
		return RankVeteran
	case level >= 10:
		return RankSeeker
	default:
		return RankRecruit
	}
}

// classForStats returns the class for the given stat levels using the argmax
// rule with the Adventurer "balanced" threshold (docs/03 §5):
//
//	dominant = max(stat_levels)
//	if (max - min) <= adventurerSpread: Adventurer
//	else: class of the dominant stat
//
// Ties for the dominant stat are broken by the canonical stat order
// (domain.AllStatKeys) so the result is deterministic.
func classForStats(levels map[domain.StatKey]int) string {
	if len(levels) == 0 {
		return ClassAdventurer
	}
	maxLvl, minLvl := math.MinInt, math.MaxInt
	var dominant domain.StatKey
	for _, k := range domain.AllStatKeys {
		lvl, ok := levels[k]
		if !ok {
			continue
		}
		if lvl > maxLvl {
			maxLvl = lvl
			dominant = k
		}
		if lvl < minLvl {
			minLvl = lvl
		}
	}
	if maxLvl-minLvl <= adventurerSpread {
		return ClassAdventurer
	}
	if c, ok := classByStat[dominant]; ok {
		return c
	}
	return ClassAdventurer
}

// RecalcClassAndRank recomputes the character's class (argmax over stat levels,
// with the "adventurer" balance threshold) and rank (level band) in place.
// See docs/03 §5 and docs/01 §4/§6.
func (e *Engine) RecalcClassAndRank(char *domain.Character) error {
	if char == nil {
		return store.ErrNotFound
	}
	stats, err := e.store.GetStats(context.Background(), char.ID)
	if err != nil {
		return err
	}
	levels := make(map[domain.StatKey]int, len(stats))
	for _, s := range stats {
		levels[s.Key] = s.Level
	}
	char.Class = classForStats(levels)
	char.Rank = RankForLevel(char.Level)
	return nil
}

// recalcClassAndRankFromLevels recomputes class/rank from already-loaded stat
// levels, returning the previous values for change detection. It avoids a
// second GetStats round-trip during Checkin.
func recalcClassAndRankFromLevels(char *domain.Character, levels map[domain.StatKey]int) (prevClass, prevRank string) {
	prevClass, prevRank = char.Class, char.Rank
	char.Class = classForStats(levels)
	char.Rank = RankForLevel(char.Level)
	return prevClass, prevRank
}

// durationMult returns the optional duration multiplier for an activity that
// tracks duration: clamp(durationMin / refMinutes, 0.5, 2.0); 1.0 otherwise
// (docs/03 §4).
func durationMult(act *domain.Activity, durationMin int) float64 {
	if !act.HasDuration || act.RefMinutes <= 0 || durationMin <= 0 {
		return 1.0
	}
	m := float64(durationMin) / float64(act.RefMinutes)
	if m < 0.5 {
		m = 0.5
	}
	if m > 2.0 {
		m = 2.0
	}
	return m
}

// dropChanceByRarity is the per-rarity item-drop probability after a check-in
// (docs/01 §3, docs/04 §4: higher rarity → lower chance).
var dropChanceByRarity = map[string]float64{
	"common":    0.40,
	"uncommon":  0.20,
	"rare":      0.10,
	"epic":      0.04,
	"legendary": 0.01,
}

// dropCapDecay returns the reward multiplier applied to a check-in that is the
// n-th of the day for an activity (1-indexed), given the activity's daily cap:
// the first `cap` check-ins pay full (1.0); each one beyond the cap halves the
// reward, flooring at 0 (docs/03 §7 anti-abuse).
func dropCapDecay(nth, cap int) float64 {
	if cap < 1 {
		cap = 1
	}
	if nth <= cap {
		return 1.0
	}
	over := nth - cap
	m := math.Pow(0.5, float64(over))
	if m < 0.0001 {
		return 0
	}
	return m
}

// Checkin records a check-in for the given character and returns the resulting
// reward event. It is the source of truth for reward math (docs/03 §4) and
// performs the whole accrual as one logical transaction (docs/07 §4).
//
// Order of accrual (docs/03 §4, docs/04 §5): base → streak → crit → duration →
// equipment, then stat points, gold (with streak bonus), drop roll, streak /
// last-checkin-date update (honouring freezes), character + stat level recalc,
// quest advancement, achievement unlocks and class/rank recalculation.
func (e *Engine) Checkin(ctx context.Context, char *domain.Character, activityKey string, durationMin int, note string) (domain.RewardEvent, error) {
	if char == nil {
		return domain.RewardEvent{}, store.ErrNotFound
	}

	act, err := e.store.GetActivity(ctx, activityKey)
	if err != nil {
		return domain.RewardEvent{}, err
	}

	now := e.now()
	localDate := dateOnly(now)

	// --- daily cap / anti-abuse (docs/03 §7) ---
	todays, err := e.store.TodayCheckins(ctx, char.ID, localDate)
	if err != nil {
		return domain.RewardEvent{}, err
	}
	priorSame := 0
	for _, k := range todays {
		if k == activityKey {
			priorSame++
		}
	}
	nth := priorSame + 1
	capDecay := dropCapDecay(nth, act.DailyCap)

	// --- streak update (docs/01 §5, docs/03 §6) ---
	// Determine the streak that applies to *this* check-in based on the gap
	// since the last check-in date. A freeze (handled by the store/inventory
	// layer) may bridge a single missed day; here we treat a 1- or 2-day gap as
	// continuation per the soft-penalty design and reset on larger gaps.
	streakDays := e.advanceStreak(char, localDate)
	streakMult := e.StreakMult(streakDays)

	// --- reward math: base → streak → crit → duration → equipment ---
	baseXP := float64(act.BaseXP)
	durMult := durationMult(act, durationMin)

	isCrit := e.randFloat() < e.balance.CritChance
	critMult := 1.0
	if isCrit {
		critMult = e.balance.CritMult
	}

	equipMult := e.equipmentXPMult(ctx, char, act.StatKey)

	xpF := baseXP * streakMult * critMult * durMult * equipMult * capDecay
	xp := int(math.Round(xpF))

	goldF := float64(act.BaseGold) * (1 + e.balance.GoldStreakBonus*(streakMult-1)) * capDecay
	gold := int(math.Round(goldF))

	statPointsF := baseXP * durMult * capDecay
	statPoints := int(math.Round(statPointsF))

	// --- apply XP / level-up (docs/03 §2) ---
	fromLevel := char.Level
	char.XPTotal += int64(xp)
	char.Level = e.LevelForXP(char.XPTotal)

	// --- apply stat points / stat level-up (docs/03 §3, §5) ---
	stats, err := e.store.GetStats(ctx, char.ID)
	if err != nil {
		return domain.RewardEvent{}, err
	}
	levels := make(map[domain.StatKey]int, len(stats))
	var statLevelUp *domain.StatLevelChange
	for i := range stats {
		st := &stats[i]
		if st.Key == act.StatKey {
			fromStatLevel := st.Level
			st.Value += int64(statPoints)
			st.Level = e.StatLevelForValue(st.Value)
			if st.Level > fromStatLevel {
				statLevelUp = &domain.StatLevelChange{Key: st.Key, From: fromStatLevel, To: st.Level}
			}
			if err := e.store.SaveStat(ctx, st); err != nil {
				return domain.RewardEvent{}, err
			}
		}
		levels[st.Key] = st.Level
	}

	// --- class / rank recalc (docs/03 §5, docs/01 §4/§6) ---
	prevClass, prevRank := recalcClassAndRankFromLevels(char, levels)

	var levelUp *domain.LevelChange
	if char.Level > fromLevel {
		levelUp = &domain.LevelChange{From: fromLevel, To: char.Level}
	}
	var rankUp *domain.RankChange
	if char.Rank != prevRank {
		rankUp = &domain.RankChange{From: prevRank, To: char.Rank}
	}
	_ = prevClass // class change is reflected on the character; not a reward-event field.

	// --- drop roll by activity rarity (docs/01 §3, docs/04 §4) ---
	drop := e.rollDrop(ctx, act)

	// --- gold accrual + ledger ---
	char.Gold += int64(gold)

	// --- persist the audit log + character + transaction ---
	logRow := &domain.ActivityLog{
		CharacterID: char.ID,
		ActivityKey: activityKey,
		DurationMin: durationMin,
		Note:        note,
		XPAwarded:   xp,
		GoldAwarded: gold,
		StatAwarded: statPoints,
		IsCrit:      isCrit,
		CreatedAt:   now,
		LocalDate:   localDate,
	}
	if err := e.store.InsertActivityLog(ctx, logRow); err != nil {
		return domain.RewardEvent{}, err
	}
	if gold != 0 {
		_ = e.store.AddTransaction(ctx, &domain.Transaction{
			CharacterID: char.ID,
			Amount:      gold,
			Reason:      "checkin",
			RefID:       activityKey,
			CreatedAt:   now,
		})
	}
	if err := e.store.SaveCharacter(ctx, char); err != nil {
		return domain.RewardEvent{}, err
	}

	// --- advance active quests (docs/02 §2) ---
	questsAdvanced := e.advanceQuests(ctx, char, act, durationMin)

	// --- unlock achievements (docs/02 §3) ---
	achievementsUnlocked := e.unlockAchievements(ctx, char, levels, fromLevel, isCrit, drop != nil)

	return domain.RewardEvent{
		Reward: domain.RewardCore{
			XP:         xp,
			Gold:       gold,
			StatKey:    act.StatKey,
			StatPoints: statPoints,
			IsCrit:     isCrit,
			StreakDays: streakDays,
			StreakMult: streakMult,
		},
		Drop:                 drop,
		LevelUp:              levelUp,
		RankUp:               rankUp,
		StatLevelUp:          statLevelUp,
		QuestsAdvanced:       questsAdvanced,
		AchievementsUnlocked: achievementsUnlocked,
		Character: domain.CharacterSummary{
			Level:      char.Level,
			XPTotal:    char.XPTotal,
			Gold:       char.Gold,
			StreakDays: char.StreakDays,
		},
	}, nil
}

// advanceStreak updates char.StreakDays / BestStreak / LastCheckinDate based on
// the gap between the last check-in date and the current local date, and
// returns the streak value that applies to this check-in (docs/01 §5).
//
//   - same day as last check-in → streak unchanged (already counted today)
//   - exactly the next day      → streak += 1
//   - a one-day gap             → streak += 1 (soft "freeze" bridge, docs/04 §6)
//   - larger gap / first check-in on a fresh streak → reset to 1
func (e *Engine) advanceStreak(char *domain.Character, localDate time.Time) int {
	if char.LastCheckinDate == nil {
		char.StreakDays = 1
	} else {
		last := dateOnly(*char.LastCheckinDate)
		gap := int(localDate.Sub(last).Hours() / 24)
		switch {
		case gap <= 0:
			// same day (or clock skew): keep current streak, at least 1.
			if char.StreakDays < 1 {
				char.StreakDays = 1
			}
		case gap == 1:
			char.StreakDays++
		case gap == 2:
			// one missed day bridged by a freeze: streak continues.
			char.StreakDays++
		default:
			char.StreakDays = 1
		}
	}
	if char.StreakDays > char.BestStreak {
		char.BestStreak = char.StreakDays
	}
	d := localDate
	char.LastCheckinDate = &d
	return char.StreakDays
}

// equipmentXPMult returns the additive equipment XP multiplier for the given
// stat, capped at +40% (docs/04 §5). With no inventory/effect data available
// through the store contract it returns 1.0; the hook is kept so equipment
// boosts apply *after* duration in the accrual order.
func (e *Engine) equipmentXPMult(ctx context.Context, char *domain.Character, stat domain.StatKey) float64 {
	if len(char.Equipped) == 0 {
		return 1.0
	}
	items, err := e.store.ListInventory(ctx, char.ID)
	if err != nil || len(items) == 0 {
		return 1.0
	}
	shop, err := e.store.ListShopItems(ctx)
	if err != nil {
		return 1.0
	}
	byID := make(map[string]domain.ShopItem, len(shop))
	for _, si := range shop {
		byID[si.ID] = si
	}
	invByID := make(map[int64]domain.InventoryItem, len(items))
	for _, it := range items {
		invByID[it.ID] = it
	}
	var bonus float64
	for _, invID := range char.Equipped {
		inv, ok := invByID[invID]
		if !ok {
			continue
		}
		si, ok := byID[inv.ShopItemID]
		if !ok {
			continue
		}
		if si.Effect.Type == "xp_mult" && (si.Effect.Stat == "" || si.Effect.Stat == stat) {
			bonus += si.Effect.Value
		}
	}
	if bonus > 0.40 {
		bonus = 0.40
	}
	return 1.0 + bonus
}

// rollDrop performs the item-drop roll for an activity by its rarity and, on
// success, returns a populated Drop by picking a shop item of the matching
// rarity (docs/01 §3, docs/04 §4). Returns nil when nothing drops.
func (e *Engine) rollDrop(ctx context.Context, act *domain.Activity) *domain.Drop {
	chance, ok := dropChanceByRarity[act.Rarity]
	if !ok {
		chance = dropChanceByRarity["common"]
	}
	if e.randFloat() >= chance {
		return nil
	}
	items, err := e.store.ListShopItems(ctx)
	if err != nil || len(items) == 0 {
		return nil
	}
	// Deterministic candidate ordering for testability.
	cands := make([]domain.ShopItem, 0, len(items))
	for _, it := range items {
		if it.Rarity == act.Rarity {
			cands = append(cands, it)
		}
	}
	if len(cands) == 0 {
		return nil
	}
	sort.Slice(cands, func(i, j int) bool { return cands[i].ID < cands[j].ID })
	pick := cands[int(e.randFloat()*float64(len(cands)))%len(cands)]
	return &domain.Drop{ItemID: pick.ID, Name: pick.Name, Rarity: pick.Rarity, Slot: pick.Slot}
}

// advanceQuests increments progress on the character's active quests whose
// condition matches THIS check-in's activity, and marks them completed when they
// reach their target (docs/02 §2). A quest only advances if its condition
// activity matches the activity (with "language" matching english|spanish).
// Increment metric depends on the condition: "minutes" quests accumulate the
// check-in duration; everything else (count/target/streak_days) advances by one.
//
// NOTE (follow-up): streak_days chains are approximated as a per-matching-check-in
// counter rather than true per-activity consecutive-day streaks, and a quest may
// advance more than once on a day with multiple matching check-ins. Proper
// per-activity day tracking (once-per-local-day via period_key) is a future refinement.
func (e *Engine) advanceQuests(ctx context.Context, char *domain.Character, act *domain.Activity, durationMin int) []domain.QuestProgress {
	qs, err := e.store.ListQuestsWithProgress(ctx, char.ID)
	if err != nil {
		return nil
	}
	var advanced []domain.QuestProgress
	for _, q := range qs {
		if q.Status != "active" {
			continue
		}
		if !questMatchesActivity(q.Condition, act) {
			continue
		}
		progress := q.Progress + questIncrement(q.Condition, act, durationMin)
		status := "active"
		var completedAt *time.Time
		if q.Target > 0 && progress >= q.Target {
			progress = q.Target
			status = "completed"
			t := e.now()
			completedAt = &t
		}
		qp := domain.QuestProgress{
			CharacterID: char.ID,
			QuestID:     q.ID,
			Progress:    progress,
			Target:      q.Target,
			Status:      status,
			CompletedAt: completedAt,
		}
		if err := e.store.UpsertQuestProgress(ctx, &qp); err != nil {
			continue
		}
		advanced = append(advanced, qp)
	}
	return advanced
}

// questMatchesActivity reports whether a quest's condition is satisfied by the
// given check-in activity. Quests without an "activity" constraint are not driven
// by check-ins here (they are level/balance/class quests evaluated elsewhere).
func questMatchesActivity(cond map[string]any, act *domain.Activity) bool {
	if cond == nil || act == nil {
		return false
	}
	raw, ok := cond["activity"]
	if !ok {
		return false
	}
	a, _ := raw.(string)
	switch a {
	case act.Key:
		return true
	case "language": // meta-group: any language activity
		return act.Key == "english" || act.Key == "spanish"
	default:
		return false
	}
}

// questIncrement returns how much a matching check-in advances a quest. Minute
// goals accumulate the check-in's duration (falling back to the activity's
// reference minutes, then 1); all other conditions advance by one.
func questIncrement(cond map[string]any, act *domain.Activity, durationMin int) int {
	if _, ok := cond["minutes"]; ok {
		switch {
		case durationMin > 0:
			return durationMin
		case act != nil && act.RefMinutes > 0:
			return act.RefMinutes
		default:
			return 1
		}
	}
	return 1
}

// unlockAchievements evaluates milestone achievements reached by this check-in
// and records the newly unlocked ones (docs/02 §3). It covers the milestone
// families that are computable from engine state: first level-up, level bands,
// crit, drop and the "all stats ≥ 10" balance achievement.
func (e *Engine) unlockAchievements(ctx context.Context, char *domain.Character, levels map[domain.StatKey]int, fromLevel int, isCrit, gotDrop bool) []string {
	all, err := e.store.ListAchievements(ctx, char.ID)
	if err != nil {
		return nil
	}
	unlockedSet := make(map[string]bool, len(all))
	for _, a := range all {
		if a.Unlocked {
			unlockedSet[a.ID] = true
		}
	}

	candidates := e.satisfiedAchievements(char, levels, fromLevel, isCrit, gotDrop)

	var newly []string
	for _, id := range candidates {
		if unlockedSet[id] {
			continue
		}
		// Only unlock IDs the catalog actually knows about.
		known := false
		for _, a := range all {
			if a.ID == id {
				known = true
				break
			}
		}
		if !known {
			continue
		}
		if err := e.store.UnlockAchievement(ctx, char.ID, id); err != nil {
			continue
		}
		unlockedSet[id] = true
		newly = append(newly, id)
	}
	return newly
}

// satisfiedAchievements returns the catalog ids of milestone achievements whose
// condition is satisfied by the current engine state. The ids follow the seed
// naming in docs/02 §3.
func (e *Engine) satisfiedAchievements(char *domain.Character, levels map[domain.StatKey]int, fromLevel int, isCrit, gotDrop bool) []string {
	var ids []string
	ids = append(ids, "first_checkin")
	if char.Level > fromLevel && fromLevel <= 1 {
		ids = append(ids, "first_levelup")
	}
	if gotDrop {
		ids = append(ids, "first_drop")
	}
	switch {
	case char.Level >= 100:
		ids = append(ids, "level_100")
		fallthrough
	case char.Level >= 50:
		ids = append(ids, "level_50")
		fallthrough
	case char.Level >= 25:
		ids = append(ids, "level_25")
		fallthrough
	case char.Level >= 10:
		ids = append(ids, "level_10")
	}
	switch {
	case char.StreakDays >= 100:
		ids = append(ids, "streak_100")
		fallthrough
	case char.StreakDays >= 30:
		ids = append(ids, "streak_30")
		fallthrough
	case char.StreakDays >= 7:
		ids = append(ids, "streak_7")
	}
	// Balance achievement: all five stats at level ≥ 10.
	if len(levels) == len(domain.AllStatKeys) {
		allTen := true
		for _, k := range domain.AllStatKeys {
			if levels[k] < 10 {
				allTen = false
				break
			}
		}
		if allTen {
			ids = append(ids, "harmony")
		}
	}
	_ = isCrit // crit-specific achievements are not in the seed set.
	return ids
}

// dateOnly truncates a time to its calendar date in its own location.
func dateOnly(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
