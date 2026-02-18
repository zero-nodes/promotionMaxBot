package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"agrotorgPromotionMaxBot/internal/db"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)

func UserOnRegistration (ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	err := db.AddUser(ctx, userId, "User on registration", 1)
	if err != nil {
		return fmt.Errorf("Error add user -> %w", err);
	}

	err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.Hello))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.RegistrationQuestion))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	return nil
}

func ContinuationRegistration (ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *Settings) error {
	maxId := upd.GetUserID();
	if upd.Message.Body.Text == "" {
		err := api.Messages.Send(ctx, maxbot.NewMessage().SetUser(maxId).SetText("Некорректные данные. Пожалуйста, введите текстовое имя"))
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err);
		}
		return nil
	}
	
	userName := upd.Message.Body.Text;
	err := db.SetUserNameAndStatusById(ctx, maxId, userName, 0);
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}
	
	
	err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(maxId).SetText("Регистрация прошла успешно, " + userName + "!"))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = SendMenu(ctx, api, upd.GetUserID(), db, cfg)
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err)
	}

	return nil
}

func SendMenu(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	err := db.SetUserStatusById(ctx, userId, 0);
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}

	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().AddCallback("Активировать купон", schemes.DEFAULT, "startActivationTiket")
	keyboard.AddRow().AddCallback("Количество Ваших активных купонов", schemes.DEFAULT, "tiketsCount")
	keyboard.AddRow().AddLink("Проверь свои шансы на победу", schemes.DEFAULT, cfg.Link.Rating)
	keyboard.AddRow().AddCallback("Правила акции", schemes.DEFAULT, "promotionRules").AddLink("🌐 Наш сайт", schemes.DEFAULT, cfg.Link.Main)

	err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText(cfg.Message.MenuText))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}
	return nil
}

func SendRules(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	err := api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.PromotionRules))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = SendMenu(ctx, api, userId, db, cfg);
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err);
	}
	return nil
}

func SendTiketsCount(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	tiketsCount, err := db.GetTiketsCount(ctx, userId);
	if err != nil {
		return fmt.Errorf("Error get tikets count -> %w", err)
	}
	
	err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.ActiveTicketsCount + fmt.Sprintf("%v", tiketsCount)))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = SendMenu(ctx, api, userId, db, cfg);
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err);
	}
	return nil
}

func StartActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	userShowAds, err := db.GetUserShowAdsById(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get user show_ads -> %w", err);
	}

	if userShowAds == true {
		keyboard := api.Messages.NewKeyboardBuilder()
		keyboard.AddRow().AddLink("Наш канал", schemes.DEFAULT, cfg.Link.Channel)
		keyboard.AddRow().AddCallback("Продлжить активацию купона", schemes.DEFAULT, "continuationActivationTiket")
		keyboard.AddRow().AddCallback("Больше не показывать", schemes.DEFAULT, "notShowAgainTextWithChannelLink")
		keyboard.AddRow().AddCallback("Вернуться в меню", schemes.DEFAULT, "menu")

		err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText(cfg.Message.TextWithChannelLink))
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err);
		}
	} else {
		err := ContinuationActivationTiket(ctx, api, userId, db, cfg);
		if err != nil {
			return fmt.Errorf("Error continuation activation tiket -> %w", err);
		}
	}

	return nil
}

func ContinuationActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *Settings) error {
	todayTiketsCount, err := db.GetTiketsCountToday(ctx, userId);
	if err != nil {
		return fmt.Errorf("Error get tikets count -> %w", err)
	}
	if todayTiketsCount >= cfg.DayActivatianLimit {
		err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.ExceedingTicketActivationLimit))
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err);
		}
		err = SendMenu(ctx, api, userId, db, cfg);
		if err != nil {
			return fmt.Errorf("Error send menu -> %w", err);
		}
		return nil
	} 
	
	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().AddCallback("Вернуться в меню", schemes.DEFAULT, "menu")

	userName, err := db.GetUserNameById(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get user name -> %w", err);
	}

	err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText(fmt.Sprintf(cfg.Message.InstructionsForActivatingTheCoupon, userName)))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = db.SetUserStatusById(ctx, userId, 2);
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}

	return nil
}

func GetActivationTiket (ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *Settings) error {
	if len(upd.Message.Body.RawAttachments) == 0 {
		err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg);
		if err != nil {
			return fmt.Errorf("Error continuation activation tiket -> %w", err);
		}
		return nil
	}	

    raw := upd.Message.Body.RawAttachments[0]
	var typeCheck struct {
		Type schemes.AttachmentType `json:"type"`
	}

	if !(json.Unmarshal(raw, &typeCheck) == nil && typeCheck.Type == schemes.AttachmentImage) {
		err := ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg);
		if err != nil {
			return fmt.Errorf("Error continuation activation tiket -> %w", err);
		}
		return nil
	}	

	var photo schemes.PhotoAttachment
	if err := json.Unmarshal(raw, &photo); err != nil {
		return fmt.Errorf("unmarshal photo error -> %w", err)
	}

	resp, err := http.Get(photo.Payload.Url)
	if err != nil {
		return fmt.Errorf("failed to download -> %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read body -> %w", err)
	}

	err = db.AddTiket(ctx, upd.GetUserID(), data)
	if err != nil {
		return fmt.Errorf("failed add tiket -> %w", err)
	}

	err = api.Messages.Send(ctx, maxbot.NewMessage().SetUser(upd.GetUserID()).SetText(cfg.Message.MessageTiketActivationReady))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = SendMenu(ctx, api, upd.GetUserID(), db, cfg);
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err);
	}

	return nil
}

func main() {
	if err := dotenv.Load(); err != nil { fmt.Println(err) }
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	defer stop()

	api, _ := maxbot.New(os.Getenv("TOKEN"))
	_, err := api.Bots.GetBot(ctx)
	if err != nil { panic(err) }

	cfg, err := NewSetting("settings.json")
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
