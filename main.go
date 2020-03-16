package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/smtp"
	"os"
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/streadway/amqp"
	"golang.org/x/net/proxy"
)

type SendRequest struct {
	ID      string
	Message string
}

type SendResponse struct {
	ID      string
	Message string
}

type RabbitResponse struct {
	Type string
	Data SendResponse
}

func main() {
	bot := createBot()

	chatId, _ := strconv.ParseInt(os.Getenv("BOT_CHAT_ID"), 10, 32)

	http.HandleFunc("/bot/send", handleRequest(bot, chatId))

	go http.ListenAndServe(":"+os.Getenv("BOT_PORT"), nil)

	rabbitConnect := getRabbitConnect()

	rabbitChannel := getRabbitChannel(rabbitConnect)

	handleBot(bot, chatId, rabbitChannel)
}

func failOnError(err error, msg string) {
	if err != nil {
		sendErrorToEmail(fmt.Sprintf("%s: %s", msg, err))

		log.Fatalf("%s: %s", msg, err)
	}
}

func logOnError(err error, msg string) {
	if err != nil {
		sendErrorToEmail(fmt.Sprintf("%s: %s", msg, err))

		log.Printf("%s: %s", msg, err)
	}
}

func sendErrorToEmail(text string) {
	from := os.Getenv("BOT_LOG_EMAIL")
	pass := os.Getenv("BOT_LOG_PASSWORD")
	to := os.Getenv("BOT_LOG_EMAIL")

	msg := "From: " + from + "\n" +
		"To: " + to + "\n" +
		"Subject: Error on telegram bot:\n\n" +
		text

	err := smtp.SendMail("smtp.gmail.com:587",
		smtp.PlainAuth("", from, pass, "smtp.gmail.com"),
		from, []string{to}, []byte(msg))

	if err != nil {
		log.Printf("Smtp error: %s", err)
		return
	}

	log.Print("Email is sent")
}

func getRabbitConnect() *amqp.Connection {
	connect, err := amqp.Dial("amqp://" +
		os.Getenv("BOT_AMQP_USER") +
		":" +
		os.Getenv("BOT_AMQP_PASSWORD") +
		"@" +
		os.Getenv("BOT_AMQP_HOST"))

	failOnError(err, "Failed to connect to RabbitMQ")

	return connect
}

func getRabbitChannel(connect *amqp.Connection) *amqp.Channel {
	ch, err := connect.Channel()

	failOnError(err, "Failed to open a channel")

	return ch
}

func createBot() *tgbotapi.BotAPI {
	client := getHttpClient()

	bot, err := tgbotapi.NewBotAPIWithClient(os.Getenv("BOT_TOKEN"), "https://api.telegram.org/bot", client)

	failOnError(err, "Error connection to bot")

	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	return bot
}

func handleRequest(bot *tgbotapi.BotAPI, chatId int64) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		message := parseBody(r.Body)

		msg := generateBotRequestMessage(formatMessage(message), message.ID, chatId)

		_, err := bot.Send(msg)

		logOnError(err, "Handle request from backend error")

		fmt.Printf("Message sended to telegram. Id: %s. Text: %s\n", message.ID, message.Message)
	}
}

func getRabbitSender(channel *amqp.Channel) func(message SendResponse) {
	return func(message SendResponse) {
		rabbitResponse := RabbitResponse{
			Type: "BOT_RESPONSE",
			Data: message,
		}

		jsonMessage, _ := json.Marshal(rabbitResponse)

		err := channel.Publish(
			"",
			os.Getenv("BOT_AMQP_QUEUE"),
			false,
			false,
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(jsonMessage),
			})

		fmt.Printf("Message sended to rabbitmq. Id: %s. Text: %s\n", message.ID, message.Message)

		logOnError(err, "Failed to publish a message to rabbitmq")
	}
}

func handleBot(bot *tgbotapi.BotAPI, chatId int64, channel *amqp.Channel) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	logOnError(err, "Get updates from telegram error")

	userId := "none"

	for update := range updates {
		msg, err := botUpdateProcessor(update, &userId, chatId, getRabbitSender(channel))

		logOnError(err, "Get update telegram processor error")

		_, err = bot.Send(msg)

		logOnError(err, "Send message to telegram error")
	}
}

func getHttpClient() *http.Client {
	dialSocksProxy, err := proxy.SOCKS5("tcp", os.Getenv("BOT_PROXY"), nil, proxy.Direct)

	failOnError(err, "Connect to proxy error")

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
		"*Message* _%s_\n",
		message.ID,
		message.Message,
	)

	return text
}

func parseBody(body io.ReadCloser) SendRequest {
	decoder := json.NewDecoder(body)

	var message SendRequest

	err := decoder.Decode(&message)

	logOnError(err, "Parse request error")

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

func botUpdateProcessor(update tgbotapi.Update, userId *string, chatId int64, rabbitSender func(message SendResponse)) (tgbotapi.MessageConfig, error) {
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

				rabbitSender(SendResponse{
					ID:      *userId,
					Message: update.Message.Text,
				})
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
