package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"agrotorgPromotionMaxBot/internal/db"
	"agrotorgPromotionMaxBot/internal/settings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
	"github.com/mew-sh/dotenv"
)


func SendMenu(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().AddCallback("Начать проверку чеков", schemes.DEFAULT, "startCheckTiket")

	err := api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText("меню:"))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}
	return nil
}

func StartCheckTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	tikets, err := db.GetTicketsForModeration(ctx, cfg.NumberTicketToCheck);
	if err != nil{
		return fmt.Errorf("Error get tickets -> %w", err)
	}
	
	if len(tikets) == 0 {
		err := api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText("Чеков для проверки нет"))
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err);
		}
		err = SendMenu(ctx, api, userId, db, cfg);
		if err != nil {
			return fmt.Errorf("Error send menu -> %w", err)
		}
		return nil
	}
	
	for _, tiket := range tikets {
		filePath := fmt.Sprintf("imege_%v", tiket.ID)
		file, err := os.Create(filePath)
		if err != nil{
			return fmt.Errorf("Error to create file -> %w", err) 
		}
		file.Write(tiket.Photo)
		file.Close()

		photo, err := api.Uploads.UploadPhotoFromFile(ctx, "./" + filePath)
		if err != nil {
			return fmt.Errorf("Erorr uploads photo -> %w", err)
		}

		keyboard := api.Messages.NewKeyboardBuilder()
		keyboard.AddRow().AddCallback("подтвердить", schemes.DEFAULT, fmt.Sprintf("confirm_%v", tiket.ID))
		keyboard.AddRow().AddCallback("откланнить", schemes.DEFAULT, fmt.Sprintf("reject_%v", tiket.ID))

		err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).AddPhoto(photo).SetUser(userId).SetText(fmt.Sprint(tiket.Date.Format("02.01.2006"))))
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err);
		}
	}

	err = SendMenu(ctx, api, userId, db, cfg);
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err)
	}

	return nil
}

func ConfirmTiket(ctx context.Context, api *maxbot.Api, upd *schemes.MessageCallbackUpdate, db *db.PgStorage, cfg *settings.Settings, idTiket int) error {
    err := db.ConfirmTiketById(ctx, idTiket)
    if err != nil {
        return fmt.Errorf("error confirm ticket: %w", err)
    }

    answer := &schemes.CallbackAnswer{
        Message: &schemes.NewMessageBody{
            Text: "✅ Подтвержден",
            Attachments: []any{},
        },
    }

    _, err = api.Messages.AnswerOnCallback(ctx, upd.Callback.CallbackID, answer)
    if err != nil {
        return fmt.Errorf("error answering callback: %w", err)
    }

	err = os.Remove(fmt.Sprintf("imege_%v", idTiket))
	if err != nil {
		return fmt.Errorf("Error remove file -> %w", err)
	}

    return nil
}

func RejectTiket(ctx context.Context, api *maxbot.Api, upd *schemes.MessageCallbackUpdate, db *db.PgStorage, cfg *settings.Settings, idTiket int) error {
	err := db.RejectTiketById(ctx, idTiket)
	if err != nil {
		return fmt.Errorf("error reject ticket: %w", err)
	}

	answer := &schemes.CallbackAnswer{
		Message: &schemes.NewMessageBody{
			Text: "❌ Отклонено",
			Attachments: []any{},
		},
	}

	_, err = api.Messages.AnswerOnCallback(ctx, upd.Callback.CallbackID, answer)
	if err != nil {
		return fmt.Errorf("error answering callback: %w", err)
	}

	err = os.Remove(fmt.Sprintf("imege_%v", idTiket))
	if err != nil {
		return fmt.Errorf("Error remove file -> %w", err)
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

	cfg, err := settings.NewSetting("settings.json")
	if err != nil { panic(err) }
	
	db, err := db.NewPgStorage(os.Getenv("DB_URL"));
	if err != nil { panic(err) }
	defer db.Close()

	fmt.Println("Started bot");
	for upd := range api.GetUpdates(ctx) {
		switch upd := upd.(type) {
		case *schemes.MessageCallbackUpdate:
			switch {
			case upd.Callback.Payload == "startCheckTiket":
				err := StartCheckTiket(ctx, api, upd.GetUserID(), db, &cfg)
				if err != nil {
					fmt.Printf("Error start check ticket -> %v\n", err)
					continue
				}
			case strings.HasPrefix(upd.Callback.Payload, "confirm_"):
				idStr := strings.TrimPrefix(upd.Callback.Payload, "confirm_")
				id, err := strconv.Atoi(idStr)
				if err != nil {
					fmt.Println("invalid id:", err)
					continue
				}

				err = ConfirmTiket(ctx, api, upd, db, &cfg, id)
				if err != nil {
					fmt.Printf("Error confirm -> %v\n", err)
				}
			case strings.HasPrefix(upd.Callback.Payload, "reject_"):
				idStr := strings.TrimPrefix(upd.Callback.Payload, "reject_")
				id, err := strconv.Atoi(idStr)
				if err != nil {
					fmt.Println("invalid id:", err)
					continue
				}

				err = RejectTiket(ctx, api, upd, db, &cfg, id)
				if err != nil {
					fmt.Printf("Error reject -> %v\n", err)
				}
			}
			
		default:
			err := SendMenu(ctx, api, upd.GetUserID(), db, &cfg)
			if err != nil {
				fmt.Printf("Error send menu -> %v\n", err)
				continue
			}
		}
	}
}

