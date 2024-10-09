package tgBot

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func getStepsMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton

	var row []tgbotapi.KeyboardButton
	for i, step := range stepsOptions {
		if step == defaultSteps {
			button := tgbotapi.NewKeyboardButton("default")
			row = append(row, button)
		} else if step != defaultSteps {
			button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%d", step))
			row = append(row, button)
		}
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
	return keyboard
}

func getModelsMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton

	var row []tgbotapi.KeyboardButton
	i := 0
	for key := range modelsOptions {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", key))
		row = append(row, button)

		// Если добавили три кнопки в ряд, создаем новый ряд
		if (i+1)%3 == 0 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем текущий ряд
		}
		i += 1
	}

	// Добавляем оставшиеся кнопки, если они есть
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}
	return keyboard
}

func getSizeMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton

	var row []tgbotapi.KeyboardButton
	i := 0
	for key := range sizeOptions {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", key))
		row = append(row, button)

		// Если добавили три кнопки в ряд, создаем новый ряд
		if (i+1)%4 == 0 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем текущий ряд
		}
		i += 1
	}

	// Добавляем оставшиеся кнопки, если они есть
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}
	return keyboard
}
