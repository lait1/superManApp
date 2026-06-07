package telegram

import (
	"encoding/json"
	"net/http"
	"strings"
)

// update is the subset of a Telegram Bot API Update we react to (docs/10 §6).
type update struct {
	Message *message `json:"message"`
}

// message is the subset of a Telegram message we read.
type message struct {
	MessageID int64  `json:"message_id"`
	Text      string `json:"text"`
	Chat      chat   `json:"chat"`
}

// chat carries the chat id we reply to (for private chats == telegram_user_id).
type chat struct {
	ID int64 `json:"id"`
}

const (
	startGreeting = "Привет! Это superMen — твой RPG-трекер привычек.\n" +
		"Отмечай дела дня, качай статы, держи стрик. Жми кнопку, чтобы открыть приложение."
	helpText = "superMen — RPG-трекер привычек.\n\n" +
		"/start — открыть приложение\n" +
		"/help — эта справка\n\n" +
		"Каждый вечер бот присылает итоги дня и напоминает о стрике."
)

// WebhookHandler returns an http.Handler that processes incoming bot updates
// (prod webhook, docs/10 §6). It answers /start with a greeting + Mini App
// button and /help with short help. webAppURL is the public Mini App URL used
// for the web_app button.
func (c *Client) WebhookHandler(webAppURL string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		defer r.Body.Close()

		var upd update
		if err := json.NewDecoder(r.Body).Decode(&upd); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// Always 200 so Telegram does not retry; we dispatch best-effort.
		w.WriteHeader(http.StatusOK)

		if upd.Message == nil || upd.Message.Chat.ID == 0 {
			return
		}

		text := strings.TrimSpace(upd.Message.Text)
		// Strip a possible @botname suffix, e.g. "/start@superMenBot".
		cmd := text
		if i := strings.IndexAny(cmd, " @"); i >= 0 {
			cmd = cmd[:i]
		}

		switch cmd {
		case "/start":
			_ = c.SendMessage(r.Context(), upd.Message.Chat.ID, startGreeting, webAppURL)
		case "/help":
			_ = c.SendMessage(r.Context(), upd.Message.Chat.ID, helpText, webAppURL)
		}
	})
}
