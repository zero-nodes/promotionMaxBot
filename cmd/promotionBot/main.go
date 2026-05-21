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
	"syscall"
	"time"

	thisApi "promotionMaxBot/internal/api"
	"promotionMaxBot/internal/db"
	"promotionMaxBot/internal/settings"

	"github.com/gorilla/mux"
	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)

const webhookPath = "/webhook"

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
	router.Use(thisApi.WithCORS)

	botAPI := thisApi.NewBotApi(ctx, api, db)
	router.HandleFunc("/send/text/alluser/", botAPI.SendTextToAllUsers).Methods("POST")
	router.HandleFunc("/send/text/user", botAPI.SendTextToUsersById).Methods("POST")
	router.HandleFunc("/getRatingTable", botAPI.GetRatingTable).Methods("GET")

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
		} else if _, ok := raw["user_id"]; ok {
			upd = &schemes.BotStartedUpdate{}
		} else {
			log.Printf("Webhook: unknown update type, body: %s", string(body))
			w.WriteHeader(http.StatusOK)
			return
		}

		if err := json.Unmarshal(body, upd); err != nil {
			log.Printf("Webhook: unmarshal into %T error: %v", upd, err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		handleUpdate(context.Background(), api, upd, db, cfg)
		w.WriteHeader(http.StatusOK)
	})

	srv := &http.Server{
		Addr:    ":8088",
		Handler: router,
	}

	go func() {
		log.Println("HTTP server started on :8088")
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

func handleUpdate(ctx context.Context, api *maxbot.Api, upd interface{}, db *db.PgStorage, cfg *settings.Settings) {
	switch upd := upd.(type) {

	case *schemes.BotStartedUpdate:
		if err := UserOnRegistration(ctx, api, upd.GetUserID(), db, cfg); err != nil {
			log.Printf("UserOnRegistration error: %v", err)
		}

	case *schemes.MessageCreatedUpdate:
		userStatus, err := db.GetUserStatusById(ctx, upd.GetUserID())
		if err != nil {
			log.Printf("GetUserStatusById error: %v", err)
			return
		}
		switch userStatus {
		case 1:
			if err := ContinuationRegistration(ctx, api, upd, db, cfg); err != nil {
				log.Printf("ContinuationRegistration error: %v", err)
			}
		case 2:
			if err := GetActivationTiket(ctx, api, upd, db, cfg); err != nil {
				log.Printf("GetActivationTiket error: %v", err)
			}
		case 0:
			if err := SendMenu(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("SendMenu error: %v", err)
			}
		default:
			if err := UserOnRegistration(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("UserOnRegistration error: %v", err)
			}
		}

	case *schemes.MessageCallbackUpdate:
		switch upd.Callback.Payload {
		case "menu":
			if err := SendMenu(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("SendMenu error: %v", err)
			}
		case "startActivationTiket":
			if err := StartActivationTiket(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("StartActivationTiket error: %v", err)
			}
		case "continuationActivationTiket":
			if err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("ContinuationActivationTiket error: %v", err)
			}
		case "checkSubscription":
			err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg)
			if err != nil {
				log.Printf("checkSubscription error: %v", err)
			}
		case "notShowAgainTextWithChannelLink":
			if err := db.SetUserShowAdsById(ctx, upd.GetUserID(), false); err != nil {
				log.Printf("SetUserShowAdsById error: %v", err)
			}
			if err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("ContinuationActivationTiket error: %v", err)
			}
		case "tiketsCount":
			if err := SendTiketsCount(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("SendTiketsCount error: %v", err)
			}
		case "promotionRules":
			if err := SendRules(ctx, api, upd.GetUserID(), db, cfg); err != nil {
				log.Printf("SendRules error: %v", err)
			}
		}
	}
}

