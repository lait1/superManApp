// This file composes the production HTTP handler: the auth-protected REST API,
// the unauthenticated Telegram bot webhook, a liveness probe and the static
// Mini App frontend (web/dist) with SPA fallback. The single Go binary serves
// everything so it can run as one Railway service (docs/13-running.md).
package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Handler builds the full production handler. webhook is the Telegram bot
// webhook handler (mounted unauthenticated at POST /bot/webhook); it may be nil
// to skip the bot route. staticDir is the directory of the built frontend
// (web/dist); when empty, no static files are served and only the API/webhook
// respond.
//
// Route precedence relies on Go 1.22+ ServeMux: the more specific "/api/" and
// "/bot/webhook" patterns win over the "/" SPA catch-all.
func (s *Server) Handler(webhook http.Handler, staticDir string) http.Handler {
	mux := http.NewServeMux()

	// REST API behind the auth middleware (docs/09 §1). The inner mux uses
	// absolute "/api/v1/..." patterns, so mounting it on the "/api/" subtree
	// matches correctly.
	mux.Handle("/api/", s.authMiddleware(s.apiMux()))

	// Telegram bot webhook (docs/10 §6). Telegram does not send initData here,
	// so this route stays outside the auth middleware.
	if webhook != nil {
		mux.Handle("POST /bot/webhook", webhook)
	}

	// Liveness probe for the platform healthcheck (Railway).
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// Static Mini App frontend with SPA fallback.
	if staticDir != "" {
		mux.Handle("/", spaHandler(staticDir))
	}

	return mux
}

// spaHandler serves files from dir. Requests that resolve to an existing file
// (JS/CSS bundles, /assets/character/*, manifest.json, favicon) are served
// as-is; anything else falls back to index.html so the client-side router
// (react-router) can take over.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	index := filepath.Join(dir, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the request path against root so "../" cannot escape dir.
		upath := r.URL.Path
		if !strings.HasPrefix(upath, "/") {
			upath = "/" + upath
		}
		full := filepath.Join(dir, filepath.Clean(upath))

		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, index)
	})
}
