package main

import (
	"context"
	"log"
	"os"

	tg "github.com/MTUCI-Pixel-Team/Picture_Generator/tgBot"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panic(err)
	}

	bot, err := tg.NewBot(os.Getenv("TG_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	context := context.Background()
	defer context.Done()

	go bot.Start(context)
	select {}
}
