// Command superMen is the single backend binary: REST API + Telegram bot +
// cron scheduler. See docs/07-architecture.md.
package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"superMen/internal/api"
	"superMen/internal/config"
	"superMen/internal/game"
	"superMen/internal/scheduler"
	"superMen/internal/store"
	"superMen/internal/store/memory"
	"superMen/internal/store/postgres"
	"superMen/internal/telegram"
)

func main() {
	cfg := config.Load()
	balance := config.DefaultBalance()

	// Select the store: in-memory by default, Postgres when DATABASE_URL is set.
	var st store.Store
	if cfg.DatabaseURL != "" {
		// NOTE: register the pgx stdlib driver on deploy (blank import) so that
		// sql.Open("pgx", ...) resolves. The skeleton keeps the driver out of
		// the build; this open will fail until the driver is wired in.
		db, err := sql.Open("pgx", cfg.DatabaseURL)
		if err != nil {
			log.Fatalf("open database: %v", err)
		}
		defer db.Close()
		st = postgres.NewStore(db)
		log.Printf("store: postgres")
	} else {
		st = memory.New()
		log.Printf("store: memory")
	}

	engine := game.New(st, balance)
	server := api.NewServer(engine, st, cfg)
	tg := telegram.NewClient(cfg.TelegramBotToken)
	sched := scheduler.New(st, tg, engine, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run the notification scheduler in the background.
	go sched.Start(ctx)

	httpServer := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           server.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("listening on :%s (env=%s)", cfg.Port, cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("http server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown: %v", err)
	}
}
