package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/streadway/amqp"
	"golang.org/x/net/proxy"
)

type SendRequest struct {
	ID      string
	Name    string
	Message string
}

type SendResponse struct {
	ID      string
	Message string
}

func main() {
	bot := createBot()

	chatId, _ := strconv.ParseInt(os.Getenv("BOT_CHAT_ID"), 10, 32)

	http.HandleFunc("/bot/send", handleRequest(bot, chatId))

	go http.ListenAndServe(os.Getenv("BOT_PORT"), nil)

	rabbitConnect := getRabbitConnect()

	rabbitChannel := getRabbitChannel(rabbitConnect)

	handleBot(bot, chatId, rabbitChannel)
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
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

		fmt.Printf("Message sended to telegram. Id: %s. Name: %s. Text: %s\n", message.ID, message.Name, message.Message)

		if err != nil {
			fmt.Println(err)
		}
	}
}

func getRabbitSender(channel *amqp.Channel) func(message SendResponse) {
	return func(message SendResponse) {
		jsonMessage, _ := json.Marshal(message)

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

		failOnError(err, "Failed to publish a message")
	}
}

func handleBot(bot *tgbotapi.BotAPI, chatId int64, channel *amqp.Channel) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	if err != nil {
		fmt.Println(err)
	}

	userId := "none"

	for update := range updates {
		msg, err := botUpdateProcessor(update, &userId, chatId, getRabbitSender(channel))

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
