package tgBot

import (
	"fmt"
	"sort"

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

func getNumberResultsMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton

	var row []tgbotapi.KeyboardButton
	for i, numberResult := range numberResultsOptions {
		if numberResult == defaultNumberResults {
			button := tgbotapi.NewKeyboardButton("default")
			row = append(row, button)
		} else if numberResult != defaultNumberResults {
			button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%d", numberResult))
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

	// Создаем слайс для ключей и сортируем их
	keys := make([]string, 0, len(modelsOptions))
	for key := range modelsOptions {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Сортируем ключи

	// Генерация кнопок в нужном порядке
	i := 0
	for _, key := range keys {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", key))
		row = append(row, button)

		// Добавляем ряд каждые три кнопки
		if (i+1)%3 == 0 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем ряд
		}
		i++
	}

	// Добавляем оставшиеся кнопки, если есть
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}

	return keyboard
}

func getSizeMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton
	var row []tgbotapi.KeyboardButton

	// Создаем слайс для ключей и сортируем их
	keys := make([]string, 0, len(sizeOptions))
	for key := range sizeOptions {
		keys = append(keys, key)
	}
	sort.Strings(keys) // Сортируем ключи

	// Генерация кнопок
	i := 0
	for _, key := range keys {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", key))
		row = append(row, button)

		// Добавляем ряд каждые четыре кнопки
		if (i+1)%4 == 0 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем ряд
		}
		i++
	}

	// Добавляем оставшиеся кнопки, если есть
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}

	return keyboard
}

func getDefaultMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton

	var row []tgbotapi.KeyboardButton
	i := 0
	for _, value := range serviceCommands {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", value))
		row = append(row, button)

		// Если добавили три кнопки в ряд, создаем новый ряд
		if (i + 1) == 2 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем текущий ряд
		}
		i += 1
	}
	keyboard = append(keyboard, row)
	return keyboard
}

func getSchedulersMarkup() [][]tgbotapi.KeyboardButton {
	var keyboard [][]tgbotapi.KeyboardButton
	var row []tgbotapi.KeyboardButton

	// Создаем слайс для ключей и сортируем их
	keys := schedulersOptions
	sort.Strings(keys) // Сортируем ключи

	// Генерация кнопок в нужном порядке
	i := 0
	for _, key := range keys {
		button := tgbotapi.NewKeyboardButton(fmt.Sprintf("%s", key))
		row = append(row, button)

		// Добавляем ряд каждые три кнопки
		if (i+1)%3 == 0 {
			keyboard = append(keyboard, row)
			row = []tgbotapi.KeyboardButton{} // Очищаем ряд
		}
		i++
	}

	// Добавляем оставшиеся кнопки, если есть
	if len(row) > 0 {
		keyboard = append(keyboard, row)
	}

	return keyboard
}
