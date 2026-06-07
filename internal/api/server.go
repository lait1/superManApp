// Package api exposes the REST API of superMen over net/http. Routes follow
// docs/09-api.md; authentication is handled by the auth middleware.
package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"superMen/internal/config"
	"superMen/internal/domain"
	"superMen/internal/game"
	"superMen/internal/store"
)

// Server holds the dependencies of the HTTP layer.
type Server struct {
	engine      *game.Engine
	store       store.Store
	cfg         config.Config
	checkinRate *rateLimiter
}

// NewServer constructs an API Server.
func NewServer(engine *game.Engine, st store.Store, cfg config.Config) *Server {
	return &Server{
		engine: engine,
		store:  st,
		cfg:    cfg,
		// Soft limit on /checkin (docs/09 §4): burst of 5, ~1 token / 2s.
		checkinRate: newRateLimiter(5, 0.5),
	}
}

// Routes registers all endpoints from docs/09 §2 on a net/http ServeMux using
// Go 1.22+ method+path patterns, wraps them with the auth middleware and
// returns the resulting handler.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/me", s.handleMe)
	mux.HandleFunc("POST /api/v1/checkin", s.handleCheckin)
	mux.HandleFunc("GET /api/v1/activities", s.handleActivities)
	mux.HandleFunc("GET /api/v1/quests", s.handleQuests)
	mux.HandleFunc("POST /api/v1/quests/{id}/claim", s.handleClaimQuest)
	mux.HandleFunc("GET /api/v1/achievements", s.handleAchievements)
	mux.HandleFunc("GET /api/v1/shop", s.handleShop)
	mux.HandleFunc("POST /api/v1/shop/{itemId}/buy", s.handleBuy)
	mux.HandleFunc("GET /api/v1/inventory", s.handleInventory)
	mux.HandleFunc("POST /api/v1/inventory/{id}/equip", s.handleEquip)
	mux.HandleFunc("POST /api/v1/inventory/{id}/unequip", s.handleUnequip)
	mux.HandleFunc("GET /api/v1/report/today", s.handleReportToday)
	mux.HandleFunc("PUT /api/v1/settings/notifications", s.handleNotificationSettings)

	return s.authMiddleware(mux)
}

// --- helpers ---

// writeJSON encodes v as JSON with the given status code.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes the unified error envelope (docs/09 §1).
func (s *Server) writeError(w http.ResponseWriter, status int, code, message string) {
	s.writeJSON(w, status, ErrorResponse{Error: ErrorBody{Code: code, Message: message}})
}

// decodeJSON reads and decodes the request body into dst. It rejects unknown
// fields and returns a non-nil error on malformed input.
func (s *Server) decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

// localToday returns the current date in the user's timezone (midnight). It is
// used for daily caps, streaks and the today screens (docs/09 §4 "день в TZ").
func localToday(u *domain.User) time.Time {
	loc := time.UTC
	if u != nil && u.Timezone != "" {
		if l, err := time.LoadLocation(u.Timezone); err == nil {
			loc = l
		}
	}
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
}

// storeError maps a store sentinel error onto an HTTP response and reports
// whether it handled the error. Handlers call it for store failures.
func (s *Server) storeError(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, store.ErrNotFound):
		s.writeError(w, http.StatusNotFound, "not_found", "resource not found")
	case errors.Is(err, store.ErrInsufficientGold):
		s.writeError(w, http.StatusConflict, "insufficient_gold", err.Error())
	default:
		s.writeError(w, http.StatusInternalServerError, "internal", "internal server error")
	}
	return true
}

// --- handlers ---

// handleMe serves GET /api/v1/me (docs/09 §3): character, stats and today's
// check-ins for the main screen.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := UserFromContext(ctx)
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	stats, err := s.store.GetStats(ctx, char.ID)
	if s.storeError(w, err) {
		return
	}

	today, err := s.store.TodayCheckins(ctx, char.ID, localToday(user))
	if s.storeError(w, err) {
		return
	}
	if today == nil {
		today = []string{}
	}

	resp := MeResponse{
		Character:     s.characterDTO(char),
		Stats:         s.statDTOs(stats),
		TodayCheckins: today,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// handleCheckin serves POST /api/v1/checkin (docs/09 §3): records a check-in via
// the engine and returns the resulting reward event. A soft rate-limit guards
// against abuse (docs/09 §4).
func (s *Server) handleCheckin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	if !s.checkinRate.Allow(strconv.FormatInt(char.ID, 10)) {
		s.writeError(w, http.StatusTooManyRequests, "rate_limited", "too many check-ins, slow down")
		return
	}

	var req CheckinRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.ActivityKey == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "activityKey is required")
		return
	}
	if req.DurationMin < 0 {
		s.writeError(w, http.StatusBadRequest, "bad_request", "durationMin must be >= 0")
		return
	}

	event, err := s.engine.Checkin(ctx, char, req.ActivityKey, req.DurationMin, req.Note)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "not_found", "unknown activity")
			return
		}
		if s.storeError(w, err) {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "internal", "could not record check-in")
		return
	}

	s.writeJSON(w, http.StatusOK, event)
}

// handleActivities serves GET /api/v1/activities: the activities catalog.
func (s *Server) handleActivities(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	acts, err := s.store.ListActivities(ctx)
	if s.storeError(w, err) {
		return
	}
	if acts == nil {
		acts = []domain.Activity{}
	}
	s.writeJSON(w, http.StatusOK, ActivitiesResponse{Activities: acts})
}

// handleQuests serves GET /api/v1/quests (docs/09 §3): active quests with the
// character's progress, grouped into daily / weekly / chains.
func (s *Server) handleQuests(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	quests, err := s.store.ListQuestsWithProgress(ctx, char.ID)
	if s.storeError(w, err) {
		return
	}

	resp := QuestsResponse{
		Daily:  []domain.QuestWithProgress{},
		Weekly: []domain.QuestWithProgress{},
		Chains: []domain.QuestWithProgress{},
	}
	for _, q := range quests {
		switch q.Type {
		case "daily":
			resp.Daily = append(resp.Daily, q)
		case "weekly":
			resp.Weekly = append(resp.Weekly, q)
		case "chain":
			resp.Chains = append(resp.Chains, q)
		default:
			// side|class|balance quests are surfaced alongside the chains list.
			resp.Chains = append(resp.Chains, q)
		}
	}
	s.writeJSON(w, http.StatusOK, resp)
}

// handleClaimQuest serves POST /api/v1/quests/{id}/claim: claims the reward of a
// completed quest.
func (s *Server) handleClaimQuest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	questID := r.PathValue("id")
	if questID == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "quest id is required")
		return
	}

	reward, err := s.store.ClaimQuest(ctx, char.ID, questID)
	if err != nil {
		// A quest that is not completed (or already claimed) is a conflict.
		if errors.Is(err, store.ErrNotFound) {
			s.writeError(w, http.StatusNotFound, "not_found", "quest not found")
			return
		}
		s.writeError(w, http.StatusConflict, "quest_not_claimable", err.Error())
		return
	}
	if reward == nil {
		reward = &domain.QuestReward{}
	}
	s.writeJSON(w, http.StatusOK, ClaimResponse{OK: true, Reward: *reward})
}

// handleAchievements serves GET /api/v1/achievements: all achievements with
// unlock state for the character.
func (s *Server) handleAchievements(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	achs, err := s.store.ListAchievements(ctx, char.ID)
	if s.storeError(w, err) {
		return
	}
	if achs == nil {
		achs = []domain.AchievementWithState{}
	}
	s.writeJSON(w, http.StatusOK, AchievementsResponse{Achievements: achs})
}

// handleShop serves GET /api/v1/shop: the shop catalog.
func (s *Server) handleShop(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	items, err := s.store.ListShopItems(ctx)
	if s.storeError(w, err) {
		return
	}
	if items == nil {
		items = []domain.ShopItem{}
	}
	s.writeJSON(w, http.StatusOK, ShopResponse{Items: items})
}

// handleBuy serves POST /api/v1/shop/{itemId}/buy (docs/09 §3): debits gold and
// adds the item to the inventory. Returns 409 when the character cannot afford
// the item.
func (s *Server) handleBuy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	itemID := r.PathValue("itemId")
	if itemID == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "item id is required")
		return
	}

	newGold, invID, err := s.store.BuyItem(ctx, char.ID, itemID)
	if err != nil {
		// ErrInsufficientGold -> 409, ErrNotFound -> 404 (handled by storeError).
		if s.storeError(w, err) {
			return
		}
		s.writeError(w, http.StatusInternalServerError, "internal", "could not buy item")
		return
	}

	// Keep the in-context character snapshot consistent with the new balance.
	char.Gold = newGold
	s.writeJSON(w, http.StatusOK, BuyResponse{OK: true, Gold: newGold, InventoryItemID: invID})
}

// handleInventory serves GET /api/v1/inventory: the character's inventory items.
func (s *Server) handleInventory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	items, err := s.store.ListInventory(ctx, char.ID)
	if s.storeError(w, err) {
		return
	}
	if items == nil {
		items = []domain.InventoryItem{}
	}
	s.writeJSON(w, http.StatusOK, InventoryResponse{Items: items})
}

// handleEquip serves POST /api/v1/inventory/{id}/equip (docs/09 §3): equips an
// inventory item into its slot and returns the new equipped map.
func (s *Server) handleEquip(w http.ResponseWriter, r *http.Request) {
	s.equipOrUnequip(w, r, true)
}

// handleUnequip serves POST /api/v1/inventory/{id}/unequip: removes an item.
func (s *Server) handleUnequip(w http.ResponseWriter, r *http.Request) {
	s.equipOrUnequip(w, r, false)
}

// equipOrUnequip is the shared body of the equip/unequip handlers.
func (s *Server) equipOrUnequip(w http.ResponseWriter, r *http.Request, equip bool) {
	ctx := r.Context()
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	invID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid inventory item id")
		return
	}

	var equipped map[string]int64
	if equip {
		equipped, err = s.store.EquipItem(ctx, char.ID, invID)
	} else {
		equipped, err = s.store.UnequipItem(ctx, char.ID, invID)
	}
	if s.storeError(w, err) {
		return
	}
	if equipped == nil {
		equipped = map[string]int64{}
	}

	// Keep the in-context character snapshot consistent.
	char.Equipped = equipped
	s.writeJSON(w, http.StatusOK, EquipResponse{OK: true, Equipped: equipped})
}

// handleReportToday serves GET /api/v1/report/today (docs/09 §3): the in-app
// daily summary for the character.
func (s *Server) handleReportToday(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, _ := UserFromContext(ctx)
	char, ok := CharacterFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no character")
		return
	}

	report, err := s.store.GetReportToday(ctx, char.ID, localToday(user))
	if s.storeError(w, err) {
		return
	}
	if report == nil {
		s.writeError(w, http.StatusInternalServerError, "internal", "empty report")
		return
	}
	s.writeJSON(w, http.StatusOK, ReportResponse{Report: *report})
}

// handleNotificationSettings serves PUT /api/v1/settings/notifications
// (docs/09 §3): updates the user's timezone and notification toggles.
func (s *Server) handleNotificationSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	user, ok := UserFromContext(ctx)
	if !ok {
		s.writeError(w, http.StatusUnauthorized, "unauthorized", "no user")
		return
	}

	var req NotificationSettingsRequest
	if err := s.decodeJSON(r, &req); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	if req.Timezone == "" {
		s.writeError(w, http.StatusBadRequest, "bad_request", "timezone is required")
		return
	}
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		s.writeError(w, http.StatusBadRequest, "bad_request", "invalid timezone")
		return
	}
	if req.DailyHour < 0 || req.DailyHour > 23 {
		s.writeError(w, http.StatusBadRequest, "bad_request", "dailyHour must be in [0,23]")
		return
	}

	prefs := domain.NotifPrefs{
		Daily:          req.Daily,
		StreakReminder: req.StreakReminder,
		Morning:        req.Morning,
		Milestone:      req.Milestone,
		DailyHour:      req.DailyHour,
	}
	if err := s.store.UpdateNotificationSettings(ctx, user.ID, req.Timezone, prefs); s.storeError(w, err) {
		return
	}

	// Reflect the change in the in-context user snapshot.
	user.Timezone = req.Timezone
	user.NotifPrefs = prefs
	s.writeJSON(w, http.StatusOK, req)
}

// --- DTO mapping ---

// characterDTO maps a domain.Character to the wire CharacterDTO, computing the
// XP-into-level / XP-to-next derived fields from the engine curves (docs/09 §3).
func (s *Server) characterDTO(ch *domain.Character) CharacterDTO {
	xpToNext := s.engine.XPToNext(ch.Level)
	xpIntoLevel := ch.XPTotal - xpAtLevelStart(s.engine, ch.Level)
	if xpIntoLevel < 0 {
		xpIntoLevel = 0
	}

	equipped := ch.Equipped
	if equipped == nil {
		equipped = map[string]int64{}
	}

	return CharacterDTO{
		Name:        ch.Name,
		Level:       ch.Level,
		XPTotal:     ch.XPTotal,
		XPToNext:    xpToNext,
		XPIntoLevel: xpIntoLevel,
		Gold:        ch.Gold,
		Class:       ch.Class,
		Rank:        ch.Rank,
		StreakDays:  ch.StreakDays,
		StreakMult:  s.engine.StreakMult(ch.StreakDays),
		Equipped:    equipped,
	}
}

// statDTOs maps the 5 stat rows to wire StatDTOs, computing into-level / to-next
// from the engine stat curve (docs/09 §3).
func (s *Server) statDTOs(stats []domain.Stat) []StatDTO {
	out := make([]StatDTO, 0, len(stats))
	for _, st := range stats {
		toNext := s.engine.StatToNext(st.Level)
		into := st.Value - statAtLevelStart(s.engine, st.Level)
		if into < 0 {
			into = 0
		}
		out = append(out, StatDTO{
			Key:       st.Key,
			Value:     st.Value,
			Level:     st.Level,
			IntoLevel: into,
			ToNext:    toNext,
		})
	}
	return out
}

// xpAtLevelStart returns the cumulative XP required to reach the start of the
// given character level: sum(XPToNext(1..level-1)). The engine's XPToNext is the
// single source of truth for the curve (docs/03 §2), so this stays correct even
// as balance constants change.
func xpAtLevelStart(e *game.Engine, level int) int64 {
	var sum int64
	for l := 1; l < level; l++ {
		sum += e.XPToNext(l)
	}
	return sum
}

// statAtLevelStart returns the cumulative points required to reach the start of
// the given stat level: sum(StatToNext(1..level-1)) (docs/03 §3).
func statAtLevelStart(e *game.Engine, level int) int64 {
	var sum int64
	for l := 1; l < level; l++ {
		sum += e.StatToNext(l)
	}
	return sum
}
