package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"agrotorgPromotionMaxBot/internal/db"
	"agrotorgPromotionMaxBot/internal/settings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)


func main() {
	if err := dotenv.Load(); err != nil { fmt.Println(err) }
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	api, _ := maxbot.New(os.Getenv("TOKEN"))
	_, err := api.Bots.GetBot(ctx)
	if err != nil { panic(err) }

	cfg, err := settings.NewSetting("settings.json")
	if err != nil { panic(err) }
	
	db, err := db.NewPgStorage(os.Getenv("DB_URL"));
	if err != nil { panic(err) }
	defer db.Close()

	fmt.Println("Started bot");
	for upd := range api.GetUpdates(ctx) {
		switch upd := upd.(type) {
		case *schemes.BotStartedUpdate: // User started bot 
			err := UserOnRegistration(ctx, api, upd.GetUserID(), db, &cfg);	
			if err != nil {
				fmt.Printf("Error registration user -> %v\n", err)
			}
		case *schemes.MessageCreatedUpdate: // User send message
			userStatus, err := db.GetUserStatusById(ctx, upd.GetUserID())
			if err != nil {
				fmt.Printf("Error get status -> %v\n", err)
				continue
			}
			switch userStatus{
			case 1: 
				err := ContinuationRegistration(ctx, api, upd, db, &cfg)
				if err != nil {
					fmt.Printf("Error registration user -> %v\n", err)
				}
			case 2: 
				err := GetActivationTiket(ctx, api, upd, db, &cfg)
				if err != nil {
					fmt.Printf("Error registration user -> %v\n", err)
				}
			case 0:
				err := SendMenu(ctx, api, upd.GetUserID(), db, &cfg)
				if err != nil {
					fmt.Printf("Error send menu -> %v\n", err)
					continue
				}
			default:
				err := UserOnRegistration(ctx, api, upd.GetUserID(), db, &cfg);	
				if err != nil {
					fmt.Printf("Error registration user -> %v\n", err)
				}
				continue
			}
		case *schemes.MessageCallbackUpdate:
			switch upd.Callback.Payload {
			case "menu":
				err := SendMenu(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error send menu -> %v\n", err)
				}
			case "startActivationTiket":
				err := StartActivationTiket(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error start activation tiket -> %v\n", err);
				}
			case "continuationActivationTiket":
				err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error continuation activation tiket -> %v\n", err);
				}
			case "notShowAgainTextWithChannelLink":
				err := db.SetUserShowAdsById(ctx, upd.GetUserID(), false);
				if err != nil {
					fmt.Printf("Error update user -> %v\n", err)
				}
				err = ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error continuation activation tiket -> %v\n", err);
				}
			case "tiketsCount":
				err := SendTiketsCount(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error send tikets count -> %v\n", err);
				}
			case "promotionRules":
				err := SendRules(ctx, api, upd.GetUserID(), db, &cfg);
				if err != nil {
					fmt.Printf("Error send rules -> %v\n", err)
				}
			}
		}
	}
}
