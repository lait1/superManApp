// Package scheduler runs the periodic notification job. On each tick it selects
// users whose local time matches their notification slot and sends a daily
// report / streak reminder via the Telegram bot. See docs/06-notifications.md §5.
package scheduler

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"superMen/internal/config"
	"superMen/internal/domain"
	"superMen/internal/game"
	"superMen/internal/store"
	"superMen/internal/telegram"
)

// Notification kinds recorded in daily_reports for idempotency (docs/08 §2).
const (
	kindDaily          = "daily"
	kindStreakReminder = "streak_reminder"
	kindMorning        = "morning"
)

// Default slot hours in the user's local time when prefs do not specify one
// (docs/06 §2). DailyHour from NotifPrefs overrides defaultDailyHour.
const (
	defaultDailyHour    = 21 // evening daily report
	streakReminderHour  = 20 // streak rescue, before the day ends
	morningHour         = 9  // optional morning quests
	streakReminderFloor = 3  // do not nag newcomers (docs/06 §4)
)

// Quiet hours: never deliver between quietStart (inclusive) and quietEnd
// (exclusive) in the user's local time (docs/06 §7).
const (
	quietStart = 0
	quietEnd   = 8
)

// Scheduler ticks on an interval and dispatches notifications.
type Scheduler struct {
	store store.Store
	tg    *telegram.Client
	eng   *game.Engine
	cfg   config.Config
}

// New constructs a Scheduler.
func New(st store.Store, tg *telegram.Client, eng *game.Engine, cfg config.Config) *Scheduler {
	return &Scheduler{store: st, tg: tg, eng: eng, cfg: cfg}
}

// Start runs the ticker loop until the context is cancelled. It is intended to
// be launched in its own goroutine. The interval is cfg.NotifyTick.
func (s *Scheduler) Start(ctx context.Context) {
	interval := s.cfg.NotifyTick
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			s.tick(ctx, now)
		}
	}
}

// tick performs one notification pass: for each notification kind it resolves
// the candidate slot hours, asks the store which users are due, builds the
// message and dispatches it, then records idempotency (docs/06 §5, §7).
func (s *Scheduler) tick(ctx context.Context, now time.Time) {
	// Daily report. The slot hour is per-user (NotifPrefs.DailyHour); we probe
	// every plausible hour and let the store filter by the user's local time.
	for hour := quietEnd; hour < 24; hour++ {
		s.dispatch(ctx, now, hour, kindDaily)
	}
	// Streak reminder — single fixed slot.
	s.dispatch(ctx, now, streakReminderHour, kindStreakReminder)
	// Morning quests — single fixed slot.
	s.dispatch(ctx, now, morningHour, kindMorning)
}

// dispatch handles one (slotHour, kind) pair.
func (s *Scheduler) dispatch(ctx context.Context, now time.Time, slotHour int, kind string) {
	users, err := s.store.UsersForNotificationSlot(ctx, now, slotHour, kind)
	if err != nil {
		log.Printf("scheduler: UsersForNotificationSlot(%d,%s): %v", slotHour, kind, err)
		return
	}
	for i := range users {
		u := users[i]
		s.notifyUser(ctx, now, &u, kind)
	}
}

// notifyUser builds and sends one notification to a single user, honoring the
// toggles, quiet hours, the per-user slot hour and idempotency.
func (s *Scheduler) notifyUser(ctx context.Context, now time.Time, u *domain.User, kind string) {
	// No chat to write to (e.g. dev device-id users): nothing to do (docs/10 §7).
	if u.TelegramUserID == nil || *u.TelegramUserID == 0 {
		return
	}

	loc := userLocation(u.Timezone)
	local := now.In(loc)

	// Quiet hours: never deliver at night (docs/06 §7).
	if hour := local.Hour(); hour >= quietStart && hour < quietEnd {
		return
	}

	// Respect the per-kind toggle and the user's preferred daily hour.
	switch kind {
	case kindDaily:
		if !u.NotifPrefs.Daily {
			return
		}
		if wantHour := dailyHour(u); local.Hour() != wantHour {
			return
		}
	case kindStreakReminder:
		if !u.NotifPrefs.StreakReminder {
			return
		}
	case kindMorning:
		if !u.NotifPrefs.Morning {
			return
		}
	default:
		return
	}

	localDate := dateOnly(local)

	ch, err := s.store.GetCharacter(ctx, u.ID)
	if err != nil {
		log.Printf("scheduler: GetCharacter(user=%d): %v", u.ID, err)
		return
	}

	report, err := s.store.GetReportToday(ctx, ch.ID, localDate)
	if err != nil {
		log.Printf("scheduler: GetReportToday(char=%d): %v", ch.ID, err)
		return
	}

	// Anti-spam: streak reminder only when there is no activity today and the
	// streak is worth saving (docs/06 §4, §7). If the day had activity the daily
	// report already covers it.
	if kind == kindStreakReminder {
		if report.HadActivity || ch.StreakDays < streakReminderFloor {
			return
		}
	}

	text := s.buildMessage(kind, ch, report)
	if text == "" {
		return
	}

	// Idempotency: claim the slot before sending so a crash mid-send does not
	// double-deliver; if already claimed, skip (docs/06 §5).
	firstTime, err := s.store.MarkReportSent(ctx, u.ID, localDate, kind)
	if err != nil {
		log.Printf("scheduler: MarkReportSent(user=%d,%s): %v", u.ID, kind, err)
		return
	}
	if !firstTime {
		return
	}

	if err := s.tg.SendMessage(ctx, *u.TelegramUserID, text, s.cfg.TelegramWebappURL); err != nil {
		log.Printf("scheduler: SendMessage(user=%d,%s): %v", u.ID, kind, err)
		return
	}
}

// buildMessage renders the bot text for a kind (docs/06 §3, §4).
func (s *Scheduler) buildMessage(kind string, ch *domain.Character, report *domain.DailyReportView) string {
	switch kind {
	case kindStreakReminder:
		return streakReminderText(ch)
	case kindMorning:
		return morningText(report)
	case kindDaily:
		if report.HadActivity {
			return dailyReportText(ch, report)
		}
		return noCheckinText(ch)
	}
	return ""
}

// dailyReportText renders the "итоги дня" message (docs/06 §3).
func dailyReportText(ch *domain.Character, r *domain.DailyReportView) string {
	var b strings.Builder
	b.WriteString("🌙 Итоги дня, superMen!\n\n")
	b.WriteString("Сегодня ты:\n")
	for _, e := range r.Entries {
		line := fmt.Sprintf("  %s  +%d XP  %s", e.Title, e.XP, statIcon(e.StatKey))
		if e.Count > 1 {
			line = fmt.Sprintf("  %s ×%d  +%d XP  %s", e.Title, e.Count, e.XP, statIcon(e.StatKey))
		}
		if e.IsCrit {
			line += "  ⚡CRIT!"
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteString(fmt.Sprintf("\n📈 Всего за день: +%d XP · 💰 +%d золота\n", r.TotalXP, r.TotalGold))
	b.WriteString(fmt.Sprintf("🔥 Стрик: %d дн. (×%s)\n", r.StreakDays, formatMult(r.StreakMult)))
	b.WriteString(fmt.Sprintf("⚔️ Уровень: %d  %s\n", r.Level, levelBar(r.XPIntoLevel, r.XPToNext)))

	if open := openQuestLines(r.OpenQuests); open != "" {
		b.WriteString("\n🎯 Незакрытые квесты:\n")
		b.WriteString(open)
	}

	b.WriteString("\n«Маленькие шаги каждый день побеждают большие рывки раз в месяц.»")
	return b.String()
}

// noCheckinText renders the "no activity today" daily message (docs/06 §3).
func noCheckinText(ch *domain.Character) string {
	var b strings.Builder
	b.WriteString("🌙 Сегодня без чек-инов.\n\n")
	if ch.StreakDays > 0 {
		b.WriteString(fmt.Sprintf("🔥 Твой стрик %d дн. под угрозой!\n", ch.StreakDays))
		b.WriteString("🧊 Заморозка спасёт серию,\n   но лучше отметить хоть одно дело.\n")
	} else {
		b.WriteString("Отметь хоть одно дело, чтобы начать стрик.\n")
	}
	return b.String()
}

// streakReminderText renders the standalone streak reminder (docs/06 §4).
func streakReminderText(ch *domain.Character) string {
	return fmt.Sprintf("🔥 Стрик %d дн. под угрозой!\n"+
		"Сегодня ещё нет ни одного чек-ина — отметь хоть одно дело, чтобы сохранить серию.",
		ch.StreakDays)
}

// morningText renders the optional morning quests message (docs/06 §2).
func morningText(r *domain.DailyReportView) string {
	var b strings.Builder
	b.WriteString("☀️ Доброе утро, superMen!\n\n")
	if open := openQuestLines(r.OpenQuests); open != "" {
		b.WriteString("Квесты на сегодня:\n")
		b.WriteString(open)
	} else {
		b.WriteString("Новый день — новые победы. Отметь первое дело!\n")
	}
	return b.String()
}

// openQuestLines renders unclaimed/active quest lines for the report.
func openQuestLines(quests []domain.QuestWithProgress) string {
	var b strings.Builder
	for _, q := range quests {
		if q.Status == "claimed" || q.Status == "expired" {
			continue
		}
		b.WriteString(fmt.Sprintf("  ☐ %s (%d/%d)\n", q.Title, q.Progress, q.Target))
	}
	return b.String()
}

// dailyHour resolves the user's preferred daily report hour, falling back to the
// default evening slot when unset (docs/06 §6).
func dailyHour(u *domain.User) int {
	h := u.NotifPrefs.DailyHour
	if h <= 0 || h > 23 {
		return defaultDailyHour
	}
	return h
}

// userLocation loads the user's timezone, defaulting to UTC on error.
func userLocation(tz string) *time.Location {
	if tz == "" {
		return time.UTC
	}
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return time.UTC
	}
	return loc
}

// dateOnly truncates a local time to midnight (the report/idempotency date).
func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// statIcon maps a stat key to its display emoji (docs/06 §3 examples).
func statIcon(k domain.StatKey) string {
	switch k {
	case domain.StatSTR:
		return "💪"
	case domain.StatINT:
		return "🧠"
	case domain.StatDIS:
		return "🎯"
	case domain.StatVIT:
		return "❤️"
	case domain.StatCHA:
		return "✨"
	}
	return ""
}

// formatMult renders a streak multiplier like "1.25".
func formatMult(m float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.2f", m), "0"), ".")
}

// levelBar renders a textual XP progress bar for the current level.
func levelBar(into, toNext int64) string {
	const width = 14
	pct := 0.0
	if toNext > 0 {
		pct = float64(into) / float64(toNext)
		if pct > 1 {
			pct = 1
		}
		if pct < 0 {
			pct = 0
		}
	}
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}
	return strings.Repeat("▓", filled) + strings.Repeat("░", width-filled) + fmt.Sprintf(" %d%%", int(pct*100))
}
