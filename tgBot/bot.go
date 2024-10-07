package tgBot

import (
	"context"
	"errors"
	"log"
	"os"
	"sync"

	gp "github.com/MTUCI-Pixel-Team/Picture_Generator/generatingPic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	tg     *tgbotapi.BotAPI
	ctx    context.Context
	cancel context.CancelFunc
}

func NewBot(token string) (*Bot, error) {
	if token == "" {
		return nil, errors.New("token is empty")
	}

	tgBot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Bot{
		tg:     tgBot,
		ctx:    ctx,
		cancel: cancel,
	}, nil
}

func (b *Bot) Start(ctx context.Context) {
	log.Println("Bot is starting")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.tg.GetUpdatesChan(u)

	var wg sync.WaitGroup
	for update := range updates {

		wsClient := gp.NewWSClient(os.Getenv("API_KEY2"))

		go wsClient.Start(ctx)

		log.Println("Client connetcted")

		if update.Message == nil {
			continue
		}

		switch update.Message.Text {
		case "/start":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Hello! I'm a bot that can generate a picture for you. Just send me a message with a description of the picture you want to get. Description must be in English and be longer than 3 characters.")
			b.tg.Send(msg)
		case "/help":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID,
				"Available commands: \n/start - start the bot \n/help - get help \n/stop - stop the bot. To generate a message, enter a description.")
			b.tg.Send(msg)
		case "/stop":
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Goodbye!")
			b.tg.Send(msg)
			b.cancel()
			wg.Wait()
			return
		default:
			log.Println("User:", update.Message.Chat.UserName, "asked:", update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Generating a picture, please wait...")
			b.tg.Send(msg)
			wg.Add(1)
			go func() {
				defer wg.Done()
				wsClient.SendMsgChan <- []byte(update.Message.Text)
				select {
				case response := <-wsClient.ReceiveMsgChan:
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(response))
					b.tg.Send(msg)
				case err := <-wsClient.ErrChan:
					log.Println(err)
				}
			}()
		}
	}
	wg.Wait()
	return
}
