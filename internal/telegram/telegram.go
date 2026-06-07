// Package telegram implements the Bot API client (sendMessage) and validation
// of Telegram WebApp initData (HMAC-SHA256). See docs/10-telegram-mini-app.md §2.
package telegram

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// authDateMaxAge is the freshness window for initData. Older payloads are
// rejected as replays (docs/10 §2, step 7).
const authDateMaxAge = 24 * time.Hour

// Validation errors returned by ValidateInitData.
var (
	// ErrInvalidInitData means the payload was malformed or the hash mismatched.
	ErrInvalidInitData = errors.New("telegram: invalid initData")
	// ErrInitDataExpired means auth_date is older than authDateMaxAge.
	ErrInitDataExpired = errors.New("telegram: initData expired")
)

// Client talks to the Telegram Bot API using the bot token.
type Client struct {
	token      string
	httpClient *http.Client
}

// NewClient constructs a Client for the given bot token.
func NewClient(token string) *Client {
	return &Client{
		token:      token,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// sendMessageRequest is the Bot API sendMessage payload (subset we use).
type sendMessageRequest struct {
	ChatID      int64        `json:"chat_id"`
	Text        string       `json:"text"`
	ParseMode   string       `json:"parse_mode,omitempty"`
	ReplyMarkup *replyMarkup `json:"reply_markup,omitempty"`
}

// replyMarkup carries the inline keyboard with the web_app button.
type replyMarkup struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

// inlineButton is one inline keyboard button. When WebApp is set it opens the
// Mini App on tap (docs/10 §5).
type inlineButton struct {
	Text   string      `json:"text"`
	WebApp *webAppInfo `json:"web_app,omitempty"`
}

// webAppInfo points an inline button at the Mini App URL.
type webAppInfo struct {
	URL string `json:"url"`
}

// apiResponse is the common Bot API envelope.
type apiResponse struct {
	OK          bool   `json:"ok"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
}

// SendMessage sends a text message to chatID with an inline web_app button that
// opens the Mini App at webAppURL (docs/06 §5, docs/10 §5). When webAppURL is
// empty the message is sent without a button.
func (c *Client) SendMessage(ctx context.Context, chatID int64, text, webAppURL string) error {
	if c == nil || c.token == "" {
		return errors.New("telegram: bot token not configured")
	}

	req := sendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	if webAppURL != "" {
		req.ReplyMarkup = &replyMarkup{
			InlineKeyboard: [][]inlineButton{{
				{Text: "Открыть superMen", WebApp: &webAppInfo{URL: webAppURL}},
			}},
		}
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("telegram: marshal sendMessage: %w", err)
	}

	endpoint := "https://api.telegram.org/bot" + c.token + "/sendMessage"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("telegram: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("telegram: sendMessage: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	var api apiResponse
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &api)
	}
	if resp.StatusCode != http.StatusOK || !api.OK {
		desc := api.Description
		if desc == "" {
			desc = strings.TrimSpace(string(raw))
		}
		return fmt.Errorf("telegram: sendMessage failed: status=%d code=%d desc=%q", resp.StatusCode, api.ErrorCode, desc)
	}
	return nil
}

// tgUser mirrors the JSON object carried in initData's `user` field.
type tgUser struct {
	ID           int64  `json:"id"`
	Username     string `json:"username"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	LanguageCode string `json:"language_code"`
}

// ValidateInitData validates a Telegram WebApp initData string against the bot
// token and, on success, returns the telegram user id and username. The
// algorithm is described in docs/10-telegram-mini-app.md §2:
//
//	secret_key = HMAC_SHA256("WebAppData", BOT_TOKEN)
//	calc_hash  = hex(HMAC_SHA256(secret_key, data_check_string))
//
// and the auth_date freshness check guards against replay.
func ValidateInitData(initData, botToken string) (telegramUserID int64, username string, err error) {
	if botToken == "" {
		return 0, "", errors.New("telegram: bot token not configured")
	}

	// 1. Parse initData as a query string into key=value pairs.
	values, err := url.ParseQuery(initData)
	if err != nil {
		return 0, "", ErrInvalidInitData
	}

	// 2. Extract the `hash` field; the rest of the pairs are sorted by key.
	hash := values.Get("hash")
	if hash == "" {
		return 0, "", ErrInvalidInitData
	}

	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "hash" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 3. data_check_string = "k1=v1\nk2=v2\n..." in ascending key order.
	var sb strings.Builder
	for i, k := range keys {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(k)
		sb.WriteByte('=')
		sb.WriteString(values.Get(k))
	}
	dataCheckString := sb.String()

	// 4. secret_key = HMAC_SHA256(key="WebAppData", message=BOT_TOKEN).
	secretMAC := hmac.New(sha256.New, []byte("WebAppData"))
	secretMAC.Write([]byte(botToken))
	secretKey := secretMAC.Sum(nil)

	// 5. calc_hash = hex(HMAC_SHA256(key=secret_key, message=data_check_string)).
	calcMAC := hmac.New(sha256.New, secretKey)
	calcMAC.Write([]byte(dataCheckString))
	calcHash := calcMAC.Sum(nil)

	// 6. Constant-time comparison against the provided hash.
	expected, decErr := hex.DecodeString(hash)
	if decErr != nil || !hmac.Equal(calcHash, expected) {
		return 0, "", ErrInvalidInitData
	}

	// 7. auth_date freshness check (replay protection).
	authDateRaw := values.Get("auth_date")
	if authDateRaw == "" {
		return 0, "", ErrInvalidInitData
	}
	authUnix, convErr := strconv.ParseInt(authDateRaw, 10, 64)
	if convErr != nil {
		return 0, "", ErrInvalidInitData
	}
	authDate := time.Unix(authUnix, 0)
	if time.Since(authDate) > authDateMaxAge {
		return 0, "", ErrInitDataExpired
	}

	// 8. Decode the `user` JSON to extract id/username.
	userRaw := values.Get("user")
	if userRaw == "" {
		return 0, "", ErrInvalidInitData
	}
	var u tgUser
	if jsonErr := json.Unmarshal([]byte(userRaw), &u); jsonErr != nil {
		return 0, "", ErrInvalidInitData
	}
	if u.ID == 0 {
		return 0, "", ErrInvalidInitData
	}

	return u.ID, u.Username, nil
}
