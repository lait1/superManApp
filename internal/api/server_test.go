package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"superMen/internal/config"
	"superMen/internal/game"
	"superMen/internal/store/memory"
)

// newTestServer builds the API handler over an in-memory store in dev mode
// (so the X-Device-Id auth fallback works without Telegram initData).
func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	st := memory.New()
	engine := game.New(st, config.DefaultBalance())
	srv := NewServer(engine, st, config.Config{Env: "dev"})
	return srv.Routes()
}

func doJSON(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("X-Device-Id", "test-device")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestCharacterSetup(t *testing.T) {
	h := newTestServer(t)

	// A fresh character is not onboarded and wears the default look.
	rec := doJSON(t, h, http.MethodGet, "/api/v1/me", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /me = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var me MeResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode /me: %v", err)
	}
	if me.Character.Onboarded {
		t.Fatalf("new character must not be onboarded")
	}
	if me.Character.Appearance.SkinTone == "" {
		t.Fatalf("new character must carry a default appearance")
	}

	// Invalid appearance ids are rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/v1/character/setup",
		`{"name":"Странник","appearance":{"bodyType":"x","skinTone":"s2","hairstyle":"short","hairColor":"dark"}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("setup with bad bodyType = %d, want 400; body: %s", rec.Code, rec.Body)
	}

	// Empty name is rejected.
	rec = doJSON(t, h, http.MethodPost, "/api/v1/character/setup",
		`{"name":"   ","appearance":{"bodyType":"a","skinTone":"s2","hairstyle":"short","hairColor":"dark"}}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("setup with blank name = %d, want 400; body: %s", rec.Code, rec.Body)
	}

	// Valid setup succeeds and persists.
	rec = doJSON(t, h, http.MethodPost, "/api/v1/character/setup",
		`{"name":"  Странник  ","appearance":{"bodyType":"b","skinTone":"s3","hairstyle":"ponytail","hairColor":"red"}}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup = %d, want 200; body: %s", rec.Code, rec.Body)
	}
	var resp CharacterSetupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode setup response: %v", err)
	}
	if !resp.OK || !resp.Character.Onboarded {
		t.Fatalf("setup response must be ok+onboarded: %+v", resp)
	}
	if resp.Character.Name != "Странник" {
		t.Fatalf("name not trimmed/saved: %q", resp.Character.Name)
	}

	rec = doJSON(t, h, http.MethodGet, "/api/v1/me", "")
	if err := json.Unmarshal(rec.Body.Bytes(), &me); err != nil {
		t.Fatalf("decode /me after setup: %v", err)
	}
	if !me.Character.Onboarded || me.Character.Name != "Странник" {
		t.Fatalf("setup not persisted: %+v", me.Character)
	}
	if me.Character.Appearance.Hairstyle != "ponytail" || me.Character.Appearance.SkinTone != "s3" {
		t.Fatalf("appearance not persisted: %+v", me.Character.Appearance)
	}
}
