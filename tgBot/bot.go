package tgBot

import (
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"sync"

	pg "github.com/prorok210/WS_Client-for_runware.ai-"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSettings struct {
	steps           int
	model           string
	width           int
	heigth          int
	state           string
	numberResults   int
	scheduler       string
	generatingMsgId int
	// Добавьте другие поля, которые могут быть полезны
}

type Bot struct {
	tg           *tgbotapi.BotAPI
	ctx          context.Context
	cancel       context.CancelFunc
	userSettings sync.Map
}

var (
	defaultModel         = "runware:100@1@1"
	defaultSteps         = 10
	defaultSize          = [2]int{512, 512}
	defaultState         = "done"
	defaultNumberResults = 1
	defaultScheduler     = "Default"
)

var serviceCommands = []string{"/start", "/help", "/models", "/steps", "/size", "/number_results", "/schedulers"}

var modelsOptions = map[string]string{
	"default":               "runware:100@1@1",
	"epicRealism":           "civitai:25694@143906",
	"JuggernautXL":          "civitai:133005@782002",
	"Realistic vision":      "civitai:4201@501240",
	"Realistic vision V6.0": "civitai:4201@501240",
	"Dream Shaper":          "civitai:4384@128713",
	"ReV Animated":          "civitai:7371@425083",
	"FLUX":                  "civitai:618692@691639",
}

var stepsOptions = []int{10, 15, 20, 30, 50, 75, 100}
var numberResultsOptions = []int{1, 2, 3, 4, 5, 10}

var schedulersOptions = []string{"Default", "DDIMScheduler", "DEISMultistepScheduler", "HeunDiscreteScheduler", "KarrasVeScheduler", "DPM++ SDE"}

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

var connectionUsers = make(map[int64]*pg.WSClient)

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
		userSettings: sync.Map{}, // Инициализируем карту
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
		if update.Message == nil {
			continue
		}
		wsClient, exists := connectionUsers[update.Message.Chat.ID]
		if !exists {
			wsClient = pg.CreateWsClient(os.Getenv("API_KEY2"), uint(update.Message.Chat.ID))
			connectionUsers[update.Message.Chat.ID] = wsClient
		}

		log.Println("Client connetcted")

		if update.Message == nil {
			continue
		}

		chatID := update.Message.Chat.ID

		loadSettings, exists := b.userSettings.Load(chatID)
		if !exists {
			// По умолчанию
			loadSettings = &UserSettings{
				steps:         defaultSteps,
				model:         defaultModel,
				state:         defaultState,
				width:         defaultSize[0],
				heigth:        defaultSize[1],
				numberResults: defaultNumberResults,
				scheduler:     defaultScheduler,
			}
			b.userSettings.Store(chatID, loadSettings)
		}
		settings := loadSettings.(*UserSettings)
		log.Println("User:", update.Message.Chat.UserName, "asked:", update.Message.Text)
		switch {
		case update.Message.Text == "/cancel":
			msg := tgbotapi.NewMessage(chatID, "Operation canceled")
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			settings.state = "waitingGenerate"
			b.userSettings.Store(chatID, settings)
		case settings.state == "done" || settings.state == "" || settings.state == "waitingGenerate":
			switch update.Message.Text {
			case "/start":
				msg := tgbotapi.NewMessage(chatID,
					"Hello! I'm a bot that can generate a picture for you. Just send me a message with a description of the picture you want to get. Description must be in English and be longer than 2 characters.")
				defaultKeyboard := getDefaultMarkup()
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
				b.tg.Send(msg)
			case "/help":
				msg := tgbotapi.NewMessage(chatID,
					"Available commands: \n"+
						"/start - restart the bot \n"+
						"/help - get help \n"+
						"/models - list of all models for generate \n"+
						"/steps - More steps - better, but longer generation\n"+
						"/size - select size of the returned image\n"+
						"/number_result - select the number of generated images\n"+
						"/schedulers - select the prototype of generation\n"+
						"/cancel - back to the start menu \n\n"+
						"To generate a message, enter a description here.")
				defaultKeyboard := getDefaultMarkup()
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
				b.tg.Send(msg)
			case "/models":
				settings.state = "showVariableModels"
				b.userSettings.Store(chatID, settings)
				handleModels(b, update.Message.Text, chatID)
			case "/steps":
				settings.state = "showVariableSteps"
				b.userSettings.Store(chatID, settings)
				handleSteps(b, update.Message.Text, chatID)
			case "/size":
				settings.state = "showVariableSize"
				b.userSettings.Store(chatID, settings)
				handleSize(b, update.Message.Text, chatID)
			case "/number_results":
				settings.state = "showVariableNumberResults"
				b.userSettings.Store(chatID, settings)
				handleNumberResults(b, update.Message.Text, chatID)
			case "/schedulers":
				settings.state = "showVariableSchedulers"
				b.userSettings.Store(chatID, settings)
				handleSchedulers(b, update.Message.Text, chatID)
			default:
				// log.Println("User:", update.Message.Chat.UserName, "asked:", update.Message.Text)
				if settings.state == "waitingGenerate" {
					msg := tgbotapi.NewMessage(chatID, "Please choose a command from the keyboard while waiting for the generation to complete.")
					b.tg.Send(msg)
					continue
				}
				if len(update.Message.Text) < 3 {
					msg := tgbotapi.NewMessage(chatID, "Description must be longer than 2 characters.")
					b.tg.Send(msg)
					continue
				}
				msg := tgbotapi.NewMessage(chatID, "Generating a picture, please wait...")
				botMsg, er := b.tg.Send(msg)
				if er != nil {
					log.Println(er)
					continue
				}
				settings.generatingMsgId = botMsg.MessageID
				b.userSettings.Store(chatID, settings)
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer func() {
						settings.state = "done"
						b.userSettings.Store(chatID, settings)
						deleteMsg := tgbotapi.DeleteMessageConfig{
							ChatID:    chatID,
							MessageID: settings.generatingMsgId,
						}
						if _, err := b.tg.Request(deleteMsg); err != nil {
							log.Printf("Failed to delete message: %v", err)
						}

						wsClient.DataInChannel.Store(false)
					}()
					settings.state = "generatingPicture"
					b.userSettings.Store(chatID, settings)
					msg := pg.ReqMessage{
						PositivePrompt: string(update.Message.Text),
						Model:          settings.model,
						Steps:          settings.steps,
						Width:          settings.width,
						Height:         settings.heigth,
						NumberResults:  settings.numberResults,
						Scheduler:      settings.scheduler,
						OutputType:     []string{"URL"},
						TaskType:       "imageInference",
						TaskUUID:       pg.GenerateUUID(),
					}
					response, err := wsClient.SendAndReceiveMsg(msg)
					if err != nil {
						log.Printf("Recieve message error: %s", err)
						msg := tgbotapi.NewMessage(chatID, "Error occurred while generating a picture. Please try again or change your settings.")
						b.tg.Send(msg)
						return
					} else {
						log.Println("RESPONSE", response, len(response))

						if response == nil || len(response) == 0 || len(response[0].Data) == 0 {
							log.Println("nil response")
							msg := tgbotapi.NewMessage(chatID, "Error occurred while generating a picture. Please try again or change your settings.")
							b.tg.Send(msg)
							return
						}
						var mediaGroup []interface{}

						for _, responseData := range response {
							if len(responseData.Err) > 0 {
								log.Println("Error response", responseData.Err)
								msg := tgbotapi.NewMessage(chatID, "Error occurred while generating a picture. Please try again or change your settings.")
								b.tg.Send(msg)
								return
							}
							imageURL := string(responseData.Data[0].ImageURL)

							resp, err := http.Get(imageURL)
							if err != nil {
								log.Println(err)
								msg := tgbotapi.NewMessage(chatID, "Failed to load image in tgChat")
								b.tg.Send(msg)
								return
							}
							defer resp.Body.Close()

							imageBytes, err := io.ReadAll(resp.Body)
							if err != nil {
								log.Println(err)
								msg := tgbotapi.NewMessage(chatID, "Error while processing image")
								b.tg.Send(msg)
								return
							}

							// Добавляем изображение в слайс как InputMediaPhoto
							photo := tgbotapi.NewInputMediaPhoto(tgbotapi.FileBytes{
								Name:  "image",
								Bytes: imageBytes,
							})

							mediaGroup = append(mediaGroup, photo)
						}

						// Отправляем все фотографии разом
						if len(mediaGroup) > 0 {
							mediaMsg := tgbotapi.NewMediaGroup(chatID, mediaGroup)

							_, err := b.tg.SendMediaGroup(mediaMsg)
							if err != nil {
								// Обрабатываем ошибку при отправке
								msg := tgbotapi.NewMessage(chatID, "Не удалось отправить изображения")
								b.tg.Send(msg)
								return
							}
						}
						// for _, responseData := range response {
						// 	if len(responseData.Err) > 0 {
						// 		log.Println("Error response", responseData.Err)
						// 		msg := tgbotapi.NewMessage(chatID, "Error occurred while generating a picture. Please try again or change your settings.")
						// 		b.tg.Send(msg)
						// 		return
						// 	}
						// 	imageURL := string(responseData.Data[0].ImageURL)

						// 	// Загружаем изображение по URL
						// 	resp, err := http.Get(imageURL)
						// 	if err != nil {
						// 		// Обрабатываем ошибку, если не удалось загрузить изображение
						// 		fmt.Println(err)
						// 		msg := tgbotapi.NewMessage(chatID, "Failed to load image in tgChat")
						// 		b.tg.Send(msg)
						// 		return
						// 	}

						// 	// Читаем изображение из ответа
						// 	imageBytes, err := io.ReadAll(resp.Body)
						// 	if err != nil {
						// 		// Обрабатываем ошибку, если не удалось прочитать изображение
						// 		fmt.Println(err)
						// 		msg := tgbotapi.NewMessage(chatID, "Error while processing image")
						// 		b.tg.Send(msg)
						// 		return
						// 	}

						// 	// Создаем объект для отправки фото
						// 	photoFileBytes := tgbotapi.FileBytes{
						// 		Name:  "image",
						// 		Bytes: imageBytes,
						// 	}
						// 	photoMsg := tgbotapi.NewPhoto(chatID, photoFileBytes)

						// 	// Отправляем изображение пользователю
						// 	_, err = b.tg.Send(photoMsg)
						// 	if err != nil {
						// 		// Обрабатываем ошибку при отправке сообщения
						// 		msg := tgbotapi.NewMessage(chatID, "Не удалось отправить изображение")
						// 		b.tg.Send(msg)
						// 		return
						// 	}
						// }
					}
				}()

			}
		case settings.state == "generatingPicture":
			msg := tgbotapi.NewMessage(chatID, "Please wait, the picture is being generated")
			b.tg.Send(msg)
		case settings.state == "chooseModels":
			handleModels(b, update.Message.Text, chatID)
		case settings.state == "chooseSteps":
			handleSteps(b, update.Message.Text, chatID)
		case settings.state == "chooseSize":
			handleSize(b, update.Message.Text, chatID)
		case settings.state == "chooseNumberResults":
			handleNumberResults(b, update.Message.Text, chatID)
		case settings.state == "chooseSchedulers":
			handleSchedulers(b, update.Message.Text, chatID)
		}
	}
	wg.Wait()
	return
}
