package api

import (
	"context"
	"net/http"
	"strings"

	"superMen/internal/domain"
	"superMen/internal/telegram"
)

// ctxKey is the unexported context key type for request-scoped values.
type ctxKey int

const (
	// userCtxKey holds the authenticated *domain.User in the request context.
	userCtxKey ctxKey = iota
	// characterCtxKey holds the authenticated user's *domain.Character.
	characterCtxKey
)

// withUser returns a copy of ctx carrying the authenticated user.
func withUser(ctx context.Context, u *domain.User) context.Context {
	return context.WithValue(ctx, userCtxKey, u)
}

// withCharacter returns a copy of ctx carrying the authenticated character.
func withCharacter(ctx context.Context, ch *domain.Character) context.Context {
	return context.WithValue(ctx, characterCtxKey, ch)
}

// UserFromContext extracts the authenticated user placed by the auth middleware.
// The boolean is false when no user is present.
func UserFromContext(ctx context.Context) (*domain.User, bool) {
	u, ok := ctx.Value(userCtxKey).(*domain.User)
	return u, ok
}

// CharacterFromContext extracts the authenticated character placed by the auth
// middleware. The boolean is false when no character is present.
func CharacterFromContext(ctx context.Context) (*domain.Character, bool) {
	ch, ok := ctx.Value(characterCtxKey).(*domain.Character)
	return ch, ok
}

// authMiddleware authenticates each request. It accepts either:
//
//	Authorization: tma <initData>   — validated via telegram.ValidateInitData
//	X-Device-Id: <uuid>             — dev fallback, only when cfg.IsDev()
//
// On success it resolves/creates the user (plus character) and stores both in
// the request context. See docs/09 §1 and docs/10 §2/§7. Any failure results in
// a 401 with the unified error envelope.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var user *domain.User

		switch {
		case strings.HasPrefix(r.Header.Get("Authorization"), "tma "):
			initData := strings.TrimPrefix(r.Header.Get("Authorization"), "tma ")
			tgID, username, err := telegram.ValidateInitData(initData, s.cfg.TelegramBotToken)
			if err != nil {
				s.writeError(w, http.StatusUnauthorized, "unauthorized", "invalid initData")
				return
			}
			u, err := s.store.GetOrCreateUserByTelegramID(ctx, tgID, username)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "internal", "could not resolve user")
				return
			}
			user = u

		case r.Header.Get("X-Device-Id") != "" && s.cfg.IsDev():
			deviceID := r.Header.Get("X-Device-Id")
			u, err := s.store.GetOrCreateUserByDeviceID(ctx, deviceID)
			if err != nil {
				s.writeError(w, http.StatusInternalServerError, "internal", "could not resolve user")
				return
			}
			user = u

		default:
			s.writeError(w, http.StatusUnauthorized, "unauthorized", "missing credentials")
			return
		}

		// Maintenance gate: while the app is closed, only the admin gets through;
		// everyone else sees the maintenance screen (the frontend keys off the
		// "maintenance" error code). Healthz and the bot webhook stay outside the
		// auth middleware, so they keep working.
		if s.cfg.MaintenanceMode && !s.cfg.IsAdmin(user.TelegramUserID) {
			s.writeError(w, http.StatusServiceUnavailable, "maintenance", "app is under maintenance")
			return
		}

		char, err := s.store.GetCharacter(ctx, user.ID)
		if err != nil {
			s.writeError(w, http.StatusInternalServerError, "internal", "could not load character")
			return
		}

		ctx = withUser(ctx, user)
		ctx = withCharacter(ctx, char)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
