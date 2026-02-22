package api

import (
	"promotionMaxBot/internal/db"
	"context"
	"net/http"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

type BotApi struct {
	ctx context.Context
	api *maxbot.Api
	db *db.PgStorage
}

func NewBotApi(ctx context.Context, api *maxbot.Api, db *db.PgStorage) *BotApi {
	return &BotApi{ctx, api, db}
}

func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
