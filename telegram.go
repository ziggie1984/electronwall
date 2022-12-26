package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/callebtc/electronwall/config"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type TelegramNotificator struct {
	ChatId int64
	Token  string
	TgBot  *tgbotapi.BotAPI
}

// var _ notificator = (*telegramNotificator)(nil)

func NewTelegramNotificator() (*TelegramNotificator, error) {
	chatId, _ := strconv.ParseInt(config.Configuration.TelegramNotification.ChatId, 10, 64)
	tgBot, err := tgbotapi.NewBotAPI(config.Configuration.TelegramNotification.Token)
	if err != nil {
		log.Println("Couldn't create telegram bot api")
		return nil, err
	}
	return &TelegramNotificator{ChatId: chatId, Token: config.Configuration.TelegramNotification.Token,
		TgBot: tgBot}, nil
}

func (t *TelegramNotificator) Notify(comment string) (err error) {

	if comment != "" {
		comment = fmt.Sprintf("Sender said: \"%s\"", comment)
	}

	// body := fmt.Sprintf("Subject: %s\n\nYou've received %d sats to your lightning address. %s",
	// 	"New lightning address payment", amount, comment)

	body := comment

	tgMessage := tgbotapi.NewMessage(t.ChatId, body)
	_, err = t.TgBot.Send(tgMessage)

	return err
}

func (t *TelegramNotificator) Target() string {
	return fmt.Sprintf("ChatId: %d", t.ChatId)
}
