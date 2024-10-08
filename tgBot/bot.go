package tgBot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	gp "github.com/MTUCI-Pixel-Team/Picture_Generator/generatingPic"
	"github.com/google/uuid"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSettings struct {
	steps  int
	model  string
	width  int
	heigth int
	state  string
	// Добавьте другие поля, которые могут быть полезны
}

type Bot struct {
	tg            *tgbotapi.BotAPI
	ctx           context.Context
	cancel        context.CancelFunc
	userSettings  map[int64]*UserSettings // Ключ — это ChatID пользователя, а значение — его настройки
	settingsMutex sync.RWMutex
}

var modelsOptions = map[string]string{
	"default":          "runware:100@1@1",
	"epicRealism":      "civitai:25694@143906",
	"2DN Pony":         "civitai:520661@933040",
	"JuggernautXL":     "civitai:133005@782002",
	"Realistic vision": "civitai:4201@501240",
	"FLUX":             "civitai:618692@691639",
}

var stepsOptions = []int{10, 20, 30, 50, 75, 100}

var sizeOptions = map[string][2]int{
	"1024x1024": {1024, 1024},
	"768x768":   {768, 768},
	"768x1152":  {768, 1152},
	"1152x864":  {1152, 864},
	"864x1152":  {864, 1152},
	"1360x768":  {1360, 768},
	"768x1360":  {768, 1360},
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
		tg:           tgBot,
		ctx:          ctx,
		cancel:       cancel,
		userSettings: make(map[int64]*UserSettings), // Инициализируем карту
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

		chatID := update.Message.Chat.ID

		b.settingsMutex.Lock()
		settings, exists := b.userSettings[chatID]
		b.settingsMutex.Unlock()
		if !exists {
			// По умолчанию
			settings = &UserSettings{
				steps:  15,
				model:  "runware:100@1@1",
				state:  "done",
				heigth: 512,
				width:  512,
			}
			b.userSettings[chatID] = settings
		}
		switch {
		case settings.state == "done" || settings.state == "":
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(chatID,
					"Hello! I'm a bot that can generate a picture for you. Just send me a message with a description of the picture you want to get. Description must be in English and be longer than 3 characters.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				b.tg.Send(msg)
			case "/help":
				msg := tgbotapi.NewMessage(chatID,
					"Available commands: \n/start - restart the bot \n/help - get help \n/models - list of all models for generate \n/steps - all variants of steps: More steps - better picture, but longer generation.\nTo generate a message, enter a description here.")
				msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
				b.tg.Send(msg)
			case "/models":
				b.settingsMutex.Lock()
				b.userSettings[chatID].state = "showVariableModels"
				b.settingsMutex.Unlock()
				handleModels(b, update.Message.Text, chatID)
			case "/steps":
				b.settingsMutex.Lock()
				b.userSettings[chatID].state = "showVariableSteps"
				b.settingsMutex.Unlock()
				handleSteps(b, update.Message.Text, chatID)
			case "/size":
				b.settingsMutex.Lock()
				b.userSettings[chatID].state = "showVariableSize"
				b.settingsMutex.Unlock()
				handleSize(b, update.Message.Text, chatID)
			default:
				log.Println("User:", update.Message.Chat.UserName, "asked:", update.Message.Text)

				if len(update.Message.Text) < 3 {
					msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Description must be longer than 3 characters.")
					b.tg.Send(msg)
					return
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Generating a picture, please wait...")
				b.tg.Send(msg)
				wg.Add(1)
				go func() {
					defer wg.Done()
					hexUUID := uuid.New().String()
					hexUUID = strings.ReplaceAll(hexUUID, "-", "")
					fmt.Println("WIDTH", b.userSettings[update.Message.Chat.ID].width)
					fmt.Println("HEIGTH", b.userSettings[update.Message.Chat.ID].heigth)
					msg := gp.Message{
						PositivePrompt: string(update.Message.Text),
						Model:          b.userSettings[update.Message.Chat.ID].model,
						Steps:          b.userSettings[update.Message.Chat.ID].steps,
						Width:          b.userSettings[update.Message.Chat.ID].width,
						Height:         b.userSettings[update.Message.Chat.ID].heigth,
						NumberResults:  1,
						OutputType:     []string{"URL"},
						TaskType:       "imageInference",
						TaskUUID:       hexUUID,
					}
					wsClient.SendMsgChan <- msg
					select {
					case response := <-wsClient.ReceiveMsgChan:
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, string(response))
						b.tg.Send(msg)
					case err := <-wsClient.ErrChan:
						log.Println(err)
						msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Error occurred while generating a picture. Please try again or change your settings.")
						b.tg.Send(msg)
					}
				}()

			}
		case settings.state == "chooseModels":
			handleModels(b, update.Message.Text, chatID)
		case settings.state == "chooseSteps":
			handleSteps(b, update.Message.Text, chatID)
		case settings.state == "chooseSize":
			handleSize(b, update.Message.Text, chatID)

		}

	}
	wg.Wait()
	return
}
