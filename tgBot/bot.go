package tgBot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	gp "github.com/MTUCI-Pixel-Team/Picture_Generator/generatingPic"
	"github.com/google/uuid"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UserSettings struct {
	steps int
	model string
	// Добавьте другие поля, которые могут быть полезны
}

type Bot struct {
	tg            *tgbotapi.BotAPI
	ctx           context.Context
	cancel        context.CancelFunc
	userSettings  map[int64]*UserSettings // Ключ — это ChatID пользователя, а значение — его настройки
	settingsMutex sync.RWMutex
	userStates    map[int64]string
}

var modelsOptions = map[string]string{
	"epiCRealism":     "civitai:25694@143906",
	"GhostMix":        "civitai:36520@76907",
	"RealisticVision": "civitai:178853@F47E942AD4",
	"DreamShaper":     "civitai:622@879DB523C3",
	"CyberRealistic":  "civitai:36520@76907",
	"AbsoluteReality": "civitai:36520@76907",
	"JuggernautXL":    "civitai:178853@F47E942AD4",
}

var stepsOptions = []int{10, 50, 100, 300, 500}

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
		userStates:   make(map[int64]string),
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
		case "/settings":
			go handleSettings(b, update, updates)
			return
		default:
			log.Println("User:", update.Message.Chat.UserName, "asked:", update.Message.Text)
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Generating a picture, please wait...")
			b.tg.Send(msg)
			wg.Add(1)
			go func() {
				defer wg.Done()
				hexUUID := uuid.New().String()
				hexUUID = strings.ReplaceAll(hexUUID, "-", "")
				msg := gp.Message{
					PositivePrompt: string(update.Message.Text),
					Model:          "runware:100@1@1",
					Steps:          100,
					Width:          512,
					Height:         512,
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
				}
			}()
		}
	}
	wg.Wait()
	return
}

func handleSettings(b *Bot, update tgbotapi.Update, updates tgbotapi.UpdatesChannel) {
	chatID := update.Message.Chat.ID
	// Блокируем доступ к настройкам для других горутин, пока мы их не обновим
	b.settingsMutex.Lock()
	settings, exists := b.userSettings[chatID]
	if !exists {
		// По умолчанию
		settings = &UserSettings{
			steps: 100,
			model: "runware:100@1@1",
		}
		b.userSettings[chatID] = settings
	}
	b.settingsMutex.Unlock()
	// Выводим пользователю текущие настройки
	text := fmt.Sprintf(`Your current settings Model: " %s ", Steps: " %d "Please choose an option. More info in \help`, settings.model, settings.steps)
	msg := tgbotapi.NewMessage(chatID, text)
	replyKeyboard := tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Models"),
			tgbotapi.NewKeyboardButton("Steps"),
		),
	)

	// Создаем сообщение с обычной клавиатурой
	msg.ReplyMarkup = replyKeyboard
	b.tg.Send(msg)

	b.settingsMutex.Lock()
	b.userStates[chatID] = "choosing_option"
	b.settingsMutex.Unlock()
	for update := range updates {
		chatID := update.Message.Chat.ID
		b.settingsMutex.Lock()
		state := b.userStates[chatID]
		b.settingsMutex.Unlock()

		if update.Message == nil {
			continue
		}

		switch state {
		case "choosing_option":
			switch update.Message.Text {
			case "Models":
				// Переходим к выбору модели
				msg := tgbotapi.NewMessage(chatID, "Please choose a model:")
				modelKeyboard := tgbotapi.NewReplyKeyboard(
					tgbotapi.NewKeyboardButtonRow(
						tgbotapi.NewKeyboardButton("runware:100@1@1"),
						tgbotapi.NewKeyboardButton("runware:200@2@2"),
					),
				)
				msg.ReplyMarkup = modelKeyboard
				b.tg.Send(msg)

				// Изменяем состояние пользователя
				b.settingsMutex.Lock()
				b.userStates[chatID] = "choosing_model"
				b.settingsMutex.Unlock()

			case "Steps":
				// Переходим к выбору количества шагов
				msg := tgbotapi.NewMessage(chatID, "Please enter the number of steps")
				var keyboard [][]tgbotapi.KeyboardButton

				var row []tgbotapi.KeyboardButton
				for i, step := range stepsOptions {
					button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%d", step))
					row = append(row, button)

					// Если добавили три кнопки в ряд, создаем новый ряд
					if (i+1)%3 == 0 {
						keyboard = append(keyboard, row)
						row = []tgbotapi.KeyboardButton{} // Очищаем текущий ряд
					}
				}

				// Добавляем оставшиеся кнопки, если они есть
				if len(row) > 0 {
					keyboard = append(keyboard, row)
				}
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboard...)
				b.tg.Send(msg)

				// Изменяем состояние пользователя
				b.settingsMutex.Lock()
				b.userStates[chatID] = "choosing_steps"
				b.settingsMutex.Unlock()
			}

		case "choosing_model":
			// Обработка выбранной модели
			model := update.Message.Text
			b.settingsMutex.Lock()
			settings, exists := b.userSettings[chatID]
			if exists {
				settings.model = model
				b.userSettings[chatID] = settings
			}
			b.userStates[chatID] = "done"
			b.settingsMutex.Unlock()

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Model set to: %s", model))
			b.tg.Send(msg)

		case "choosing_steps":
			// Строку в число (ascii to integer)
			steps, err := strconv.Atoi(update.Message.Text)
			// Проверка на вхождение введенного числа в список доступных значений
			ok := 0
			for _, v := range stepsOptions {
				if v == steps {
					ok = 1
				}
			}
			if err == nil && ok == 1 {
				b.settingsMutex.Lock()
				settings, exists := b.userSettings[chatID]
				if exists {
					settings.steps = steps
					b.userSettings[chatID] = settings
				}
				b.userStates[chatID] = "done"
				b.settingsMutex.Unlock()

				msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Steps set to: %d", steps))
				b.tg.Send(msg)
			} else {
				msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a number from keyboard.")
				var keyboard [][]tgbotapi.KeyboardButton

				var row []tgbotapi.KeyboardButton
				for i, step := range stepsOptions {
					button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%d", step))
					row = append(row, button)

					// Если добавили три кнопки в ряд, создаем новый ряд
					if (i+1)%3 == 0 {
						keyboard = append(keyboard, row)
						row = []tgbotapi.KeyboardButton{} // Очищаем текущий ряд
					}
				}

				// Добавляем оставшиеся кнопки, если они есть
				if len(row) > 0 {
					keyboard = append(keyboard, row)
				}
				msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboard...)
				b.tg.Send(msg)
			}
		}
	}
}
