package main

import (
	"agrotorgPromotionMaxBot/internal/db"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"agrotorgPromotionMaxBot/internal/settings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

func UserOnRegistration (ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
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

func ContinuationRegistration (ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *settings.Settings) error {
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

func SendMenu(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	err := db.SetUserStatusById(ctx, userId, 0);
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}

	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().AddCallback("Активировать купон", schemes.DEFAULT, "startActivationTiket")
	keyboard.AddRow().AddCallback("Количество Ваших активных купонов", schemes.DEFAULT, "tiketsCount")
	keyboard.AddRow().AddLink("Проверь свои шансы на победу", schemes.DEFAULT, cfg.Link.Rating)
	keyboard.AddRow().AddCallback("Правила акции", schemes.DEFAULT, "promotionRules").AddLink("Наш канал", schemes.DEFAULT, cfg.Link.Channel)

	err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText(cfg.Message.MenuText))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}
	return nil
}

func SendRules(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	err := api.Messages.Send(ctx, maxbot.NewMessage().SetUser(userId).SetText(cfg.Message.PromotionRules).SetFormat("html"))
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err);
	}

	err = SendMenu(ctx, api, userId, db, cfg);
	if err != nil {
		return fmt.Errorf("Error send menu -> %w", err);
	}
	return nil
}

func SendTiketsCount(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
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

func StartActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	userShowAds, err := db.GetUserShowAdsById(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get user show_ads -> %w", err);
	}

	if userShowAds == true {
		keyboard := api.Messages.NewKeyboardBuilder()
		keyboard.AddRow().AddLink("Перейти в канал", schemes.DEFAULT, cfg.Link.Channel)
		keyboard.AddRow().AddCallback("Продолжить активацию купона", schemes.DEFAULT, "continuationActivationTiket")
		keyboard.AddRow().AddCallback("Больше не показывать", schemes.DEFAULT, "notShowAgainTextWithChannelLink")
		keyboard.AddRow().AddCallback("Вернуться в главное меню", schemes.DEFAULT, "menu")

		err = api.Messages.Send(ctx, maxbot.NewMessage().AddKeyboard(keyboard).SetUser(userId).SetText(cfg.Message.TextWithChannelLink).SetFormat("html"))
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

func ContinuationActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
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
	keyboard.AddRow().AddCallback("Вернуться в главное меню", schemes.DEFAULT, "menu")

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

func GetActivationTiket (ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *settings.Settings) error {
	if len(upd.Message.Body.RawAttachments) != 1 {
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

