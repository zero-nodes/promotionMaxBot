package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"promotionMaxBot/internal/db"
	"promotionMaxBot/internal/settings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)

func main() {
	if err := dotenv.Load(); err != nil { 
		panic(err) 
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	api, _ := maxbot.New(os.Getenv("TOKEN"))
	_, err := api.Bots.GetBot(ctx)
	if err != nil {
		panic(err)
	}

	cfg, err := settings.NewSetting("settings.json")
	if err != nil { 
		panic(err)
	}
	
	db, err := db.NewPgStorage(os.Getenv("DB_URL"));
	if err != nil { 
		panic(err)
	}
	defer db.Close()

	log.Println("Started bot");
	for upd := range api.GetUpdates(ctx) {
		switch upd := upd.(type) {
		case *schemes.MessageCallbackUpdate:
			switch {
			case upd.Callback.Payload == "startCheckTiket":
				err := StartCheckTiket(ctx, api, upd.GetUserID(), db, &cfg)
				if err != nil {
					log.Printf("Error start check ticket -> %v\n", err)
					continue
				}
			case strings.HasPrefix(upd.Callback.Payload, "confirm_"):
				idStr := strings.TrimPrefix(upd.Callback.Payload, "confirm_")
				id, err := strconv.Atoi(idStr)
				if err != nil {
					log.Println("invalid id:", err)
					continue
				}

				err = ConfirmTiket(ctx, api, upd, db, &cfg, id)
				if err != nil {
					log.Printf("Error confirm -> %v\n", err)
				}
			case strings.HasPrefix(upd.Callback.Payload, "reject_"):
				idStr := strings.TrimPrefix(upd.Callback.Payload, "reject_")
				id, err := strconv.Atoi(idStr)
				if err != nil {
					log.Println("invalid id:", err)
					continue
				}

				err = RejectTiket(ctx, api, upd, db, &cfg, id)
				if err != nil {
					log.Printf("Error reject -> %v\n", err)
				}
			}
			
		default:
			err := SendMenu(ctx, api, upd.GetUserID(), db, &cfg)
			if err != nil {
				log.Printf("Error send menu -> %v\n", err)
				continue
			}
		}
	}
}

