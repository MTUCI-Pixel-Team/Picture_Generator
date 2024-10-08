package tgBot

import (
	"fmt"
	"log"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func handleSteps(b *Bot, message string, chatID int64) {

	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	switch {
	case settings.state == "showVariableSteps":

		// Переходим к выбору количества шагов
		text := fmt.Sprintf(`Your current settings steps: "%d" Please choose one from keyboard. More info in /help`, settings.steps)
		msg := tgbotapi.NewMessage(chatID, text)

		keyboardSteps := getStepsMarkup()

		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboardSteps...)
		b.tg.Send(msg)
		settings.state = "chooseSteps"
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = settings.state
		b.settingsMutex.Unlock()
	case settings.state == "chooseSteps":
		// Строку в число (ascii to integer)
		steps, err := strconv.Atoi(message)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please choose a number from keyboard:")
			b.tg.Send(msg)
			return
		}

		// Проверка на вхождение введенного числа в список доступных значений
		ok := false
		for _, v := range stepsOptions {
			if v == steps {
				ok = true
				break
			}
		}
		if !ok {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a number from keyboard.")
			b.tg.Send(msg)
		} else {
			settings.steps = steps
			settings.state = "done"
			b.settingsMutex.Lock()
			b.userSettings[chatID].steps = settings.steps
			b.userSettings[chatID].state = settings.state
			b.settingsMutex.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Steps set to: %d", steps))
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			b.tg.Send(msg)
			return
		}
	}

}

func handleModels(b *Bot, message string, chatID int64) {
	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	fmt.Println("STATE", settings.state)
	switch settings.state {
	case "showVariableModels":
		modelName := ""
		for key, value := range modelsOptions {
			if value == settings.model {
				modelName = key
				break
			}
		}
		// Переходим к выбору количества шагов
		text := fmt.Sprintf(`Your model: "%s" Please choose one from keyboard. More info in /help`, modelName)
		msg := tgbotapi.NewMessage(chatID, text)
		keyboard := getModelsMarkup()
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboard...)
		b.tg.Send(msg)
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = "chooseModels"
		b.settingsMutex.Unlock()
	case "chooseModels":
		// Проверка на вхождение введенной модели в список доступных значений
		ok := 0
		for key := range modelsOptions {
			if key == message {
				ok = 1
			}
		}
		if ok == 1 {
			b.settingsMutex.Lock()
			b.userSettings[chatID].model = modelsOptions[message]
			b.userSettings[chatID].state = "done"
			b.settingsMutex.Unlock()

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Model set to: %s", message))
			msg.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
			b.tg.Send(msg)
			return
		} else if ok == 0 {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a model from keyboard.")
			b.tg.Send(msg)
		}

	}
}
