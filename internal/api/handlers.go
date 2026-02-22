package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	maxbot "github.com/max-messenger/max-bot-api-client-go"
)

func GetAuthorizationToken(r *http.Request) (string, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return "", fmt.Errorf("error get auth token")
	}

	buf := strings.Fields(auth)
	if len(buf) != 2 || strings.ToLower(buf[0]) != "bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return buf[1], nil
}

func (api *BotApi) SendTextToAllUsers (w http.ResponseWriter, r *http.Request) {
	token, err := GetAuthorizationToken(r)
	if err != nil {
		log.Println("error api send text to all users ->", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if token != os.Getenv("THIS_API_TOKEN") {
		log.Println("error api send text to all users ->", "incorrect token")
		http.Error(w, "incorrect token", http.StatusUnauthorized)
		return
	}
	
	var jsonBodyMap map[string]any
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("error api send text to all users ->", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &jsonBodyMap)
	if err != nil {
		log.Println("error api send text to all users ->", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgVal, ok := jsonBodyMap["message"]
	if !ok {
		http.Error(w, "message not found", http.StatusBadRequest)
		return
	}

	textMessage, ok := msgVal.(string)
	if !ok {
		http.Error(w, "message must be string", http.StatusBadRequest)
		return
	}

	userIdList, err := api.db.GetListAllUserId(api.ctx)
	if err != nil {
		log.Println("error api send text to all users ->", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, userId := range(userIdList) {
		err = api.api.Messages.Send(api.ctx, maxbot.NewMessage().SetUser(userId).SetText(textMessage))
		if err != nil {
			log.Println("error send message to", userId, "->", err)
		}
	}

	w.WriteHeader(http.StatusOK)
}

func (api *BotApi) SendTextToUsersById (w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.ParseInt(idStr, 10, 64)   
    if err != nil {
		log.Println("error send text to user by id ->", err)
        http.Error(w, "Invalid id", http.StatusBadRequest)
        return
    }

	token, err := GetAuthorizationToken(r)
	if err != nil {
		log.Println("error api send text to user by is ->", err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if token != os.Getenv("THIS_API_TOKEN") {
		log.Println("error api send text to user by id ->", "incorrect token")
		http.Error(w, "incorrect token", http.StatusUnauthorized)
		return
	}
	
	var jsonBodyMap map[string]any
	body, err := io.ReadAll(r.Body)
	err = json.Unmarshal(body, &jsonBodyMap)
	if err != nil {
		log.Println("error api send text to user by id ->", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	msgVal, ok := jsonBodyMap["message"]
	if !ok {
		http.Error(w, "message not found", http.StatusBadRequest)
		return
	}

	textMessage, ok := msgVal.(string)
	if !ok {
		http.Error(w, "message must be string", http.StatusBadRequest)
		return
	}

	err = api.api.Messages.Send(api.ctx, maxbot.NewMessage().SetUser(id).SetText(textMessage))
	if err != nil {
		log.Println("error send message to", id, "->", err)
	}
	
	w.WriteHeader(http.StatusOK)
}

func (api *BotApi) GetRatingTable (w http.ResponseWriter, r *http.Request) {
	rating, err := api.db.GetRating(api.ctx);
	if err != nil {
		log.Println("error get rating ->", err)
        http.Error(w, "error get rating", http.StatusInternalServerError)
        return
	}

	jsonData, err := json.Marshal(rating)
	if err != nil {
		log.Println("error marshal json ->", err)
        http.Error(w, "error marshal json", http.StatusInternalServerError)
        return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}
