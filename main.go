package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path"

	"github.com/gorilla/mux"
)

type Notification struct {
	Message string `json:"message"`
}

var Config = make(map[string]interface{})

type Message struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

func main() {
	Start()
}

func FailOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func SendApiError(w http.ResponseWriter, err error, httpStatus int) {
	if err != nil {
		httpErr := map[string]interface{}{
			"error": err.Error(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(httpStatus)

		json.NewEncoder(w).Encode(httpErr)
	}
}

func init() {
	Config["APP_PORT"] = 5551
	Config["TG_CHAT_ID"] = ""   // your telegram chat id, get it on https://t.me/getmyid_bot
	Config["TG_BOT_TOKEN"] = "" // your bot token, get it on https://t.me/BotFather
	Config["TOKEN"] = ""        // api auth token, you can set it as anything
	Config["TG_API_BOT_BASE_URL"] = fmt.Sprintf("https://api.telegram.org/bot%s/", Config["TG_BOT_TOKEN"])
}

func getSendMessageURL() string {
	baseURL, err := url.Parse(Config["TG_API_BOT_BASE_URL"].(string))
	FailOnError(err)

	baseURL.Path = path.Join(baseURL.Path, "/sendMessage")

	return baseURL.String()
}

func sendMessage(message Message) (*http.Response, error) {
	body, err := json.Marshal(message)
	FailOnError(err)

	return http.Post(getSendMessageURL(), "application/json", bytes.NewReader(body))
}

func authenticatedReq(r *http.Request) bool {
	return r.Header.Get("token") == Config["TOKEN"]
}

func sendNotificationHandler(w http.ResponseWriter, r *http.Request) {
	if authenticatedReq(r) == false {
		SendApiError(w, errors.New("invalid token"), http.StatusForbidden)
		return
	}

	var notification Notification
	err := json.NewDecoder(r.Body).Decode(&notification)
	if err != nil {
		SendApiError(w, errors.New("invalid request body: cannot parse body to Notification object"), http.StatusBadRequest)
		return
	}

	if notification.Message == "" {
		SendApiError(w, errors.New("empty message passed"), http.StatusBadRequest)
		return
	}

	msg := Message{
		ChatID: Config["TG_CHAT_ID"].(string),
		Text:   notification.Message,
	}

	telegramResponse, err := sendMessage(msg)
	SendApiError(w, err, http.StatusInternalServerError)

	if telegramResponse.StatusCode == http.StatusOK {
		res := map[string]interface{}{
			"message": "Notification has been sent.",
		}

		ReturnResponse(w, res, http.StatusOK)
		SendApiError(w, err, http.StatusInternalServerError)
	} else {
		res := map[string]interface{}{
			"message": "Notification cannot be sent.",
			"error":   err.Error(),
		}

		err = ReturnResponse(w, res, http.StatusBadRequest)
		SendApiError(w, err, http.StatusInternalServerError)
	}
}

func Start() {
	router := mux.NewRouter()
	router.HandleFunc("/send-notification", sendNotificationHandler).Methods("POST")

	log.Printf("Listening on localhost:%v", Config["APP_PORT"])
	err := http.ListenAndServe(fmt.Sprintf(":%v", Config["APP_PORT"]), router)
	FailOnError(err)
}

func ReturnResponse(w http.ResponseWriter, res map[string]interface{}, httpStatus int) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := json.NewEncoder(w).Encode(res)

	return err
}
