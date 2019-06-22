package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"golang.org/x/net/proxy"
)

type SendRequest struct {
	ID      string
	Name    string
	Message string
}

func main() {
	bot := createBot()

	chatId, _ := strconv.ParseInt(os.Getenv("BOT_CHAT_ID"), 10, 32)

	http.HandleFunc("/bot/send", handleRequest(bot, chatId))

	go http.ListenAndServe(os.Getenv("BOT_PORT"), nil)

	handleBot(bot, chatId)
}

func createBot() *tgbotapi.BotAPI {
	client := getHttpClient()

	bot, err := tgbotapi.NewBotAPIWithClient(os.Getenv("BOT_TOKEN"), client)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	return bot
}

func handleRequest(bot *tgbotapi.BotAPI, chatId int64) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		message := parseBody(r.Body)

		msg := generateBotRequestMessage(formatMessage(message), message.ID, chatId)

		_, err := bot.Send(msg)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func handleBot(bot *tgbotapi.BotAPI, chatId int64) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		fmt.Println(err)
	}

	userId := "none"

	for update := range updates {
		msg, err := botUpdateProcessor(update, &userId, chatId, func() {
			// Todo rabbit
		})

		if err != nil {
			continue
		}

		_, err = bot.Send(msg)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func getHttpClient() *http.Client {
	dialSocksProxy, err := proxy.SOCKS5("tcp", os.Getenv("BOT_PROXY"), nil, proxy.Direct)

	if err != nil {
		fmt.Println(err)
	}

	transport := &http.Transport{
		Dial: dialSocksProxy.Dial,
	}

	client := &http.Client{
		Transport: transport,
	}

	return client
}

func formatMessage(message SendRequest) string {
	text := fmt.Sprintf("*Новое сообщение*\n"+
		"*ID* _%s_\n"+
		"*Name* _%s_\n"+
		"*Message* _%s_\n",
		message.ID,
		message.Name,
		message.Message,
	)

	return text
}

func parseBody(body io.ReadCloser) SendRequest {
	decoder := json.NewDecoder(body)

	var message SendRequest

	err := decoder.Decode(&message)

	if err != nil {
		fmt.Println(err)
	}

	return message
}

func generateBotRequestMessage(message string, requestId string, chatId int64) tgbotapi.MessageConfig {
	msg := tgbotapi.NewMessage(chatId, message)
	msg.ParseMode = "markdown"

	keyboard := tgbotapi.InlineKeyboardMarkup{}

	var row []tgbotapi.InlineKeyboardButton

	responseButton := tgbotapi.NewInlineKeyboardButtonData("Ответить", requestId)
	byeButton := tgbotapi.NewInlineKeyboardButtonData("Закрыть", "none")

	row = append(row, responseButton, byeButton)

	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, row)

	msg.ReplyMarkup = keyboard

	return msg
}

func botUpdateProcessor(update tgbotapi.Update, userId *string, chatId int64, rabbitSender func()) (tgbotapi.MessageConfig, error) {
	var msg tgbotapi.MessageConfig

	switch {
	case update.Message != nil:
		{
			if update.Message.Chat.ID != chatId {
				return msg, &tgbotapi.Error{Message: "Permission denied"}
			}

			if *userId == "none" {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Некому писать")
			} else {
				msg = tgbotapi.NewMessage(update.Message.Chat.ID, "Вы ответили пользователю с ID "+*userId)

				rabbitSender()
			}

			return msg, nil
		}
	case update.CallbackQuery != nil:
		{
			*userId = update.CallbackQuery.Data

			var msg tgbotapi.MessageConfig

			if *userId == "none" {
				msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Вы закрыли активный диалог")
			} else {
				msg = tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, "Вы открыли диалог с ID "+*userId)
			}

			return msg, nil
		}
	}

	return msg, &tgbotapi.Error{Message: "Empty message"}
}
