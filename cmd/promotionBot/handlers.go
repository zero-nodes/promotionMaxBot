package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"promotionMaxBot/internal/db"
	"promotionMaxBot/internal/settings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
	"github.com/max-messenger/max-bot-api-client-go/schemes"
)

// UserOnRegistration регистрация нового пользователя
func UserOnRegistration(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	err := db.AddUser(ctx, userId, "User on registration", 1)
	if err != nil {
		return fmt.Errorf("Error add user -> %w", err)
	}

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(userId).
			SetText(cfg.Message.Hello),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(userId).
			SetText(cfg.Message.RegistrationQuestion),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}

	return nil
}

// ContinuationRegistration завершение регистрации
func ContinuationRegistration(ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *settings.Settings) error {
	maxId := upd.GetUserID()

	if upd.Message.Body.Text == "" {
		err := api.Messages.Send(ctx,
			maxbot.NewMessage().
				SetUser(maxId).
				SetText("Некорректные данные. Пожалуйста, введите текстовое имя"),
		)
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err)
		}
		return nil
	}

	userName := upd.Message.Body.Text
	err := db.SetUserNameAndStatusById(ctx, maxId, userName, 0)
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(maxId).
			SetText("Регистрация прошла успешно, "+userName+"!"),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}

	return SendMenu(ctx, api, upd.GetUserID(), db, cfg)
}

// SendMenu главное меню
func SendMenu(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	err := db.SetUserStatusById(ctx, userId, 0)
	if err != nil {
		return fmt.Errorf("Error update user -> %w", err)
	}

	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback("Активировать купон", schemes.DEFAULT, "startActivationTiket")
	keyboard.AddRow().
		AddCallback("Количество Ваших активных купонов", schemes.DEFAULT, "tiketsCount")
	keyboard.AddRow().
		AddLink("Проверь свои шансы на победу", schemes.DEFAULT, cfg.Link.Rating)
	keyboard.AddRow().
		AddCallback("Правила акции", schemes.DEFAULT, "promotionRules").
		AddLink("Наш канал", schemes.DEFAULT, cfg.Link.Channel)

	return api.Messages.Send(ctx,
		maxbot.NewMessage().
			AddKeyboard(keyboard).
			SetUser(userId).
			SetText(cfg.Message.MenuText),
	)
}

// SendRules отправка правил акции
func SendRules(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	err := api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(userId).
			SetText(cfg.Message.PromotionRules).
			SetFormat("html"),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}
	return SendMenu(ctx, api, userId, db, cfg)
}

// SendTiketsCount количество активных купонов
func SendTiketsCount(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	tiketsCount, err := db.GetTiketsCount(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get tikets count -> %w", err)
	}

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(userId).
			SetText(cfg.Message.ActiveTicketsCount+fmt.Sprintf("%v", tiketsCount)),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}
	return SendMenu(ctx, api, userId, db, cfg)
}

// StartActivationTiket – немедленная проверка подписки (старая логика с рекламой удалена)
func StartActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	return ContinuationActivationTiket(ctx, api, userId, db, cfg)
}

// ContinuationActivationTiket проверяет подписку и, если успешно, запускает активацию купона
func ContinuationActivationTiket(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	channelID := cfg.Link.ChannelID
	if channelID == 0 {
		log.Println("WARNING: ChannelID is not set, subscription check skipped")
		return activateCoupon(ctx, api, userId, db, cfg)
	}

	members, err := api.Chats.GetSpecificChatMembers(ctx, channelID, []int64{userId})
	if err != nil {
		log.Printf("GetSpecificChatMembers error: %v", err)
		return sendSubscriptionRequired(ctx, api, userId, cfg)
	}

	for _, member := range members.Members {
		if member.UserId == userId {
			return activateCoupon(ctx, api, userId, db, cfg)
		}
	}

	return sendSubscriptionRequired(ctx, api, userId, cfg)
}

// activateCoupon – основная логика выдачи купона (без изменений)
func activateCoupon(ctx context.Context, api *maxbot.Api, userId int64, db *db.PgStorage, cfg *settings.Settings) error {
	todayTiketsCount, err := db.GetTiketsCountToday(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get tikets count -> %w", err)
	}

	if todayTiketsCount >= cfg.DayActivatianLimit {
		err = api.Messages.Send(ctx,
			maxbot.NewMessage().
				SetUser(userId).
				SetText(cfg.Message.ExceedingTicketActivationLimit),
		)
		if err != nil {
			return fmt.Errorf("Send message error -> %w", err)
		}
		return SendMenu(ctx, api, userId, db, cfg)
	}

	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddCallback("Вернуться в главное меню", schemes.DEFAULT, "menu")

	userName, err := db.GetUserNameById(ctx, userId)
	if err != nil {
		return fmt.Errorf("Error get user name -> %w", err)
	}

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			AddKeyboard(keyboard).
			SetUser(userId).
			SetText(fmt.Sprintf(cfg.Message.InstructionsForActivatingTheCoupon, userName)),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}

	return db.SetUserStatusById(ctx, userId, 2)
}

// sendSubscriptionRequired – сообщение о необходимости подписки с кнопками «Перейти в канал» и «Проверить подписку»
func sendSubscriptionRequired(ctx context.Context, api *maxbot.Api, userId int64, cfg *settings.Settings) error {
	text := cfg.Message.SubscribeRequiredText
	if text == "" {
		text = "Для продолжения необходимо подписаться на наш канал."
	}

	btnLinkText := cfg.Message.SubscribeButtonText
	if btnLinkText == "" {
		btnLinkText = "Перейти в канал"
	}

	keyboard := api.Messages.NewKeyboardBuilder()
	keyboard.AddRow().
		AddLink(btnLinkText, schemes.DEFAULT, cfg.Link.Channel)
	keyboard.AddRow().
		AddCallback("Проверить подписку", schemes.DEFAULT, "checkSubscription")

	return api.Messages.Send(ctx,
		maxbot.NewMessage().
			AddKeyboard(keyboard).
			SetUser(userId).
			SetText(text),
	)
}

// GetActivationTiket – загрузка купона (без изменений)
func GetActivationTiket(ctx context.Context, api *maxbot.Api, upd *schemes.MessageCreatedUpdate, db *db.PgStorage, cfg *settings.Settings) error {
	if len(upd.Message.Body.RawAttachments) != 1 {
		return ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg)
	}

	raw := upd.Message.Body.RawAttachments[0]
	var typeCheck struct {
		Type schemes.AttachmentType `json:"type"`
	}

	if !(json.Unmarshal(raw, &typeCheck) == nil && typeCheck.Type == schemes.AttachmentImage) {
		return ContinuationActivationTiket(ctx, api, upd.GetUserID(), db, cfg)
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

	err = api.Messages.Send(ctx,
		maxbot.NewMessage().
			SetUser(upd.GetUserID()).
			SetText(cfg.Message.MessageTiketActivationReady),
	)
	if err != nil {
		return fmt.Errorf("Send message error -> %w", err)
	}

	return SendMenu(ctx, api, upd.GetUserID(), db, cfg)
}
