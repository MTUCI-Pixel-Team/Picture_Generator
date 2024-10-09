package tgBot

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	gp "github.com/MTUCI-Pixel-Team/Picture_Generator/generatingPic"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSettings struct {
	steps         int
	model         string
	width         int
	heigth        int
	state         string
	numberResults int
	// Добавьте другие поля, которые могут быть полезны
}

type Bot struct {
	tg            *tgbotapi.BotAPI
	ctx           context.Context
	cancel        context.CancelFunc
	userSettings  map[int64]*UserSettings // Ключ — это ChatID пользователя, а значение — его настройки
	settingsMutex sync.RWMutex
}

var (
	defaultModel         = "runware:100@1@1"
	defaultSteps         = 10
	defaultSize          = [2]int{512, 512}
	defaultState         = "done"
	defaultNumberResults = 1
)

var serviceCommands = []string{"/start", "/help", "/models", "/steps", "/size"}

var modelsOptions = map[string]string{
	"default":          "runware:100@1@1",
	"epicRealism":      "civitai:25694@143906",
	"2DN Pony":         "civitai:520661@933040",
	"JuggernautXL":     "civitai:133005@782002",
	"Realistic vision": "civitai:4201@501240",
	"FLUX":             "civitai:618692@691639",
}

var stepsOptions = []int{10, 15, 20, 30, 50, 75, 100}
var numberResultsOptions = []int{1, 2, 3, 4, 5, 10}

var sizeOptions = map[string][2]int{
	"default 512x512 (1:1)": {512, 512},
	"1024x1024 (1:1)":       {1024, 1024},
	"768x768 (1:1)":         {768, 768},
	"2048x2048 (1:1)":       {2048, 2048},
	"768x512 (3:2)":         {768, 512},
	"1920x1280 (3:2)":       {1920, 1280},
	"2048x1536 (4:3)":       {2048, 1536},
	"1024x768 (4:3)":        {1024, 768},
	"1536x2048 (3:4)":       {1536, 2048},
	"768x1024 (3:4)":        {768, 1024},
	"2048x1152 (16:9)":      {2048, 1152},
	"1024x1792 (9:16)":      {1024, 1792},
}

var connectionUsers = make(map[int64]*gp.WSClient)

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

func (b *Bot) Start() {
	log.Println("Bot is starting")
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	b.tg.Buffer = 100

	updates := b.tg.GetUpdatesChan(u)

	var wg sync.WaitGroup
	for update := range updates {
		// fmt.Println("update", update)

		wsClient, exists := connectionUsers[update.Message.Chat.ID]
		if !exists {
			wsClient = gp.NewWSClient(os.Getenv("API_KEY2"), uint(update.Message.Chat.ID))
			connectionUsers[update.Message.Chat.ID] = wsClient
		}

		go wsClient.Start()

		log.Println("Client connetcted")

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		b.settingsMutex.Lock()
		settings, exists := b.userSettings[chatID]
		if !exists {
			// По умолчанию
			settings = &UserSettings{
				steps:         defaultSteps,
				model:         defaultModel,
				state:         defaultState,
				width:         defaultSize[0],
				heigth:        defaultSize[1],
				numberResults: defaultNumberResults,
			}
			b.userSettings[chatID] = settings
		}
		b.settingsMutex.Unlock()
		switch {
		case update.Message.Text == "/cancel":
			msg := tgbotapi.NewMessage(chatID, "Operation canceled")
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			b.userSettings[chatID].state = "done"
		case settings.state == "done" || settings.state == "":
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(chatID,
					"Hello! I'm a bot that can generate a picture for you. Just send me a message with a description of the picture you want to get. Description must be in English and be longer than 3 characters.")
				defaultKeyboard := getDefaultMarkup()
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
				b.tg.Send(msg)
				// b.tg.StopReceivingUpdates()
				// updates = b.tg.GetUpdatesChan(u)
				// break
			case "/help":
				msg := tgbotapi.NewMessage(chatID,
					"Available commands: \n"+
						"/start - restart the bot \n"+
						"/help - get help \n"+
						"/models - list of all models for generate \n"+
						"/steps - More steps - better, but longer generation\n"+
						"/size - select size of the returned image\n"+
						"/cancel - back to the start menu \n\n"+
						"To generate a message, enter a description here.")
				defaultKeyboard := getDefaultMarkup()
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
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
			case "/numberResults":
				b.settingsMutex.Lock()
				b.userSettings[chatID].state = "showVariableNumberResults"
				b.settingsMutex.Unlock()
				handleNumberResults(b, update.Message.Text, chatID)
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
					b.settingsMutex.Lock()
					msg := gp.ReqMessage{
						PositivePrompt: string(update.Message.Text),
						Model:          b.userSettings[update.Message.Chat.ID].model,
						Steps:          b.userSettings[update.Message.Chat.ID].steps,
						Width:          b.userSettings[update.Message.Chat.ID].width,
						Height:         b.userSettings[update.Message.Chat.ID].heigth,
						NumberResults:  b.userSettings[update.Message.Chat.ID].numberResults,
						OutputType:     []string{"URL"},
						TaskType:       "imageInference",
						TaskUUID:       gp.GenerateUUID(),
					}
					b.settingsMutex.Unlock()
					wsClient.SendMsgChan <- msg
					select {
					case response := <-wsClient.ReceiveMsgChan:
						imageURL := string(response) // Предполагается, что это URL изображения
						// Загружаем изображение по URL
						resp, err := http.Get(imageURL)
						if err != nil {
							// Обрабатываем ошибку, если не удалось загрузить изображение
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось загрузить изображение")
							b.tg.Send(msg)
							return
						}
						defer resp.Body.Close()

						// Читаем изображение из ответа
						imageBytes, err := io.ReadAll(resp.Body)
						if err != nil {
							// Обрабатываем ошибку, если не удалось прочитать изображение
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Ошибка при обработке изображения")
							b.tg.Send(msg)
							return
						}

						// Создаем объект для отправки фото
						photoFileBytes := tgbotapi.FileBytes{
							Name:  "image",
							Bytes: imageBytes,
						}
						photoMsg := tgbotapi.NewPhoto(update.Message.Chat.ID, photoFileBytes)

						// Отправляем изображение пользователю
						_, err = b.tg.Send(photoMsg)
						if err != nil {
							// Обрабатываем ошибку при отправке сообщения
							msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Не удалось отправить изображение")
							b.tg.Send(msg)
							return
						}
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
		case settings.state == "chooseNumberResults":
			handleNumberResults(b, update.Message.Text, chatID)

		}

	}
	wg.Wait()
	return
}
