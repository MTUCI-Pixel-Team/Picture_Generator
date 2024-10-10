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
		text := fmt.Sprintf(`Your current settings steps: "%d" Please choose one from keyboard. Type /cancel if you want to return to the start menu `, settings.steps)
		msg := tgbotapi.NewMessage(chatID, text)

		keyboardSteps := getStepsMarkup()

		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboardSteps...)
		b.tg.Send(msg)
		settings.state = "chooseSteps"
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = settings.state
		b.settingsMutex.Unlock()
	case settings.state == "chooseSteps":
		if message == "default" {
			settings.steps = defaultSteps
			settings.state = "done"
			b.settingsMutex.Lock()
			b.userSettings[chatID].steps = settings.steps
			b.userSettings[chatID].state = settings.state
			b.settingsMutex.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Steps set to: %d", defaultSteps))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		}
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
		if ok || steps == defaultSteps {
			settings.steps = steps
			settings.state = "done"
			b.settingsMutex.Lock()
			b.userSettings[chatID].steps = settings.steps
			b.userSettings[chatID].state = settings.state
			b.settingsMutex.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Steps set to: %d", steps))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		} else {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a number from keyboard.")
			b.tg.Send(msg)
		}
	}

}

func handleNumberResults(b *Bot, message string, chatID int64) {

	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	switch {
	case settings.state == "showVariableNumberResults":

		// Переходим к выбору количества шагов
		text := fmt.Sprintf(`Your current settings number results: "%d" Please choose one from keyboard. Type /cancel if you want to return to the start menu `, settings.numberResults)
		msg := tgbotapi.NewMessage(chatID, text)

		keyboardSteps := getNumberResultsMarkup()

		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboardSteps...)
		b.tg.Send(msg)
		settings.state = "chooseNumberResults"
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = settings.state
		b.settingsMutex.Unlock()
	case settings.state == "chooseNumberResults":
		if message == "default" {
			settings.numberResults = defaultNumberResults
			settings.state = "done"
			b.settingsMutex.Lock()
			b.userSettings[chatID].numberResults = settings.numberResults
			b.userSettings[chatID].state = settings.state
			b.settingsMutex.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Number results set to: %d", defaultNumberResults))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		}
		// Строку в число (ascii to integer)
		numberResults, err := strconv.Atoi(message)
		if err != nil {
			log.Println(err)
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please choose a number from keyboard:")
			b.tg.Send(msg)
			return
		}

		// Проверка на вхождение введенного числа в список доступных значений
		ok := false
		for _, v := range numberResultsOptions {
			if v == numberResults {
				ok = true
				break
			}
		}
		if ok || numberResults == defaultSteps {
			settings.numberResults = numberResults
			settings.state = "done"
			b.settingsMutex.Lock()
			b.userSettings[chatID].numberResults = settings.numberResults
			b.userSettings[chatID].state = settings.state
			b.settingsMutex.Unlock()
			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Number results set to: %d", numberResults))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		} else {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a number from keyboard.")
			b.tg.Send(msg)
		}
	}

}

func handleModels(b *Bot, message string, chatID int64) {
	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	// fmt.Println("STATE", settings.state)
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
		text := fmt.Sprintf(`Your model: "%s" Please choose one from keyboard. Type /cancel if you want to return to the start menu`, modelName)
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
		if ok == 1 || modelsOptions[message] == defaultModel {
			b.settingsMutex.Lock()
			b.userSettings[chatID].model = modelsOptions[message]
			b.userSettings[chatID].state = "done"
			b.settingsMutex.Unlock()

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Model set to: %s", message))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		} else if ok == 0 {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a model from keyboard.")
			b.tg.Send(msg)
		}

	}
}

func handleSize(b *Bot, message string, chatID int64) {
	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	switch settings.state {
	case "showVariableSize":
		sizeName := ""
		for key, value := range sizeOptions {
			if value[0] == settings.width && value[1] == settings.heigth {
				sizeName = key
				break
			}
		}
		// Переходим к выбору количества шагов
		text := fmt.Sprintf(`Your size: "%s" Please choose one from keyboard. Type /cancel if you want to return to the start menus`, sizeName)
		msg := tgbotapi.NewMessage(chatID, text)
		keyboard := getSizeMarkup()
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboard...)
		b.tg.Send(msg)
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = "chooseSize"
		b.settingsMutex.Unlock()
	case "chooseSize":
		// Проверка на вхождение введенной модели в список доступных значений
		ok := 0
		for key := range sizeOptions {
			if key == message {
				ok = 1
			}
		}
		if ok == 1 || sizeOptions[message] == defaultSize {
			b.settingsMutex.Lock()
			b.userSettings[chatID].width = sizeOptions[message][0]
			b.userSettings[chatID].heigth = sizeOptions[message][1]
			b.userSettings[chatID].state = "done"
			b.settingsMutex.Unlock()

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Size set to: %s", message))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		} else if ok == 0 {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a model from keyboard.")
			b.tg.Send(msg)
		}

	}
}

func handleSchedulers(b *Bot, message string, chatID int64) {
	b.settingsMutex.Lock()
	settings := b.userSettings[chatID]
	b.settingsMutex.Unlock()
	// fmt.Println("STATE", settings.state)
	switch settings.state {
	case "showVariableSchedulers":
		schedulerName := ""
		for _, value := range schedulersOptions {
			if value == settings.scheduler {
				schedulerName = value
				break
			}
		}
		// Переходим к выбору количества шагов
		text := fmt.Sprintf(`Your scheduler: "%s" Please choose one from keyboard. Type /cancel if you want to return to the start menu`, schedulerName)
		msg := tgbotapi.NewMessage(chatID, text)
		keyboard := getSchedulersMarkup()
		msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(keyboard...)
		b.tg.Send(msg)
		b.settingsMutex.Lock()
		b.userSettings[chatID].state = "chooseSchedulers"
		b.settingsMutex.Unlock()
	case "chooseSchedulers":
		// Проверка на вхождение введенной модели в список доступных значений
		ok := 0
		for _, value := range schedulersOptions {
			if value == message {
				ok = 1
			}
		}
		if ok == 1 || message == defaultModel {
			b.settingsMutex.Lock()
			b.userSettings[chatID].scheduler = message
			b.userSettings[chatID].state = "done"
			b.settingsMutex.Unlock()

			msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Scheduler set to: %s", message))
			defaultKeyboard := getDefaultMarkup()
			msg.ReplyMarkup = tgbotapi.NewReplyKeyboard(defaultKeyboard...)
			b.tg.Send(msg)
			return
		} else if ok == 0 {
			msg := tgbotapi.NewMessage(chatID, "Invalid input. Please enter a scheduler from keyboard.")
			b.tg.Send(msg)
		}

	}
}
