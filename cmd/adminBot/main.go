package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"promotionMaxBot/internal/db"
	"promotionMaxBot/internal/settings"

	"github.com/gorilla/mux"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)

const webhookPath = "/webhook-admin"

var updateTypes = []string{
	"bot_started",
	"message_created",
	"message_callback",
}

func main() {
	if err := dotenv.Load(); err != nil {
		panic(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	api, err := maxbot.New(os.Getenv("TOKEN"))
	if err != nil {
		panic(err)
	}
	if _, err = api.Bots.GetBot(ctx); err != nil {
		panic(err)
	}

	cfgVal, err := settings.NewSetting("settings.json")
	if err != nil {
		panic(err)
	}
	cfg := &cfgVal

	db, err := db.NewPgStorage(os.Getenv("DB_URL"))
	if err != nil {
		panic(err)
	}
	defer db.Close()

	secret := os.Getenv("WEBHOOK_SECRET")
	publicURL := os.Getenv("PUBLIC_URL")
	webhookFullURL := publicURL + webhookPath

	subs, err := api.Subscriptions.GetSubscriptions(ctx)
	if err != nil {
		log.Printf("Warning: get subscriptions: %v", err)
	} else {
		for _, s := range subs.Subscriptions {
			if _, err := api.Subscriptions.Unsubscribe(ctx, s.Url); err != nil {
				log.Printf("Warning: unsubscribe from %s: %v", s.Url, err)
			}
		}
	}

	if _, err := api.Subscriptions.Subscribe(ctx, webhookFullURL, updateTypes, secret); err != nil {
		panic(fmt.Errorf("subscribe failed: %w", err))
	}
	log.Printf("Subscribed to webhook: %s", webhookFullURL)

	router := mux.NewRouter()

	router.HandleFunc(webhookPath, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Max-Bot-Api-Secret") != secret {
			log.Println("Webhook: invalid secret")
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			log.Printf("Webhook: read body error: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		var raw map[string]interface{}
		if err := json.Unmarshal(body, &raw); err != nil {
			log.Printf("Webhook: JSON unmarshal error: %v", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		var upd schemes.UpdateInterface
		if _, ok := raw["callback"]; ok {
			upd = &schemes.MessageCallbackUpdate{}
		} else if _, ok := raw["message"]; ok {
			upd = &schemes.MessageCreatedUpdate{}
		} else {
			upd = &schemes.BotStartedUpdate{}
		}

		if err := json.Unmarshal(body, upd); err != nil {
			log.Printf("Webhook: unmarshal into %T error: %v", upd, err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		processUpdate(context.Background(), api, upd, db, cfg)
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":8089",
		Handler: router,
	}

	go func() {
		log.Println("HTTP server started on :8089")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	if _, err := api.Subscriptions.Unsubscribe(context.Background(), webhookFullURL); err != nil {
		log.Printf("Unsubscribe error: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("HTTP server Shutdown error: %v", err)
	}
	log.Println("Bot stopped cleanly.")
}

func processUpdate(ctx context.Context, api *maxbot.Api, upd interface{}, db *db.PgStorage, cfg *settings.Settings) {
	switch upd := upd.(type) {
	case *schemes.MessageCallbackUpdate:
		switch {
		case upd.Callback.Payload == "startCheckTiket":
			if err := StartCheckTiket(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("Error start check ticket -> %v\n", err)
			}
		case strings.HasPrefix(upd.Callback.Payload, "confirm_"):
			idStr := strings.TrimPrefix(upd.Callback.Payload, "confirm_")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Println("invalid id:", err)
				return
			}
			if err = ConfirmTiket(ctx, api, upd, db, cfg, id); err != nil {
				log.Printf("Error confirm -> %v\n", err)
			}
		case strings.HasPrefix(upd.Callback.Payload, "reject_"):
			idStr := strings.TrimPrefix(upd.Callback.Payload, "reject_")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				log.Println("invalid id:", err)
				return
			}
			if err = RejectTiket(ctx, api, upd, db, cfg, id); err != nil {
				log.Printf("Error reject -> %v\n", err)
			}
		}

	case *schemes.BotStartedUpdate:
		if err := SendMenu(ctx, api, upd.GetUserID(), db, cfg); err != nil {
			log.Printf("Error send menu -> %v\n", err)
		}

	case *schemes.MessageCreatedUpdate:
		if err := SendMenu(ctx, api, upd.GetUserID(), db, cfg); err != nil {
			log.Printf("Error send menu -> %v\n", err)
		}

	default:
		log.Printf("Unknown update type: %T", upd)
	}
}
