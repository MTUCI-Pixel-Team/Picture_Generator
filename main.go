package main

import (
	"log"
	"os"
	"strconv"
	"time"

	gp "github.com/MTUCI-Pixel-Team/Picture_Generator/generatingPic"

	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Panic(err)
	}

	// bot, err := tg.NewBot(os.Getenv("TG_TOKEN"))
	// if err != nil {
	// 	log.Panic(err)
	// }

	// go bot.Start()

	wsClient := gp.NewWSClient(os.Getenv("API_KEY2"), 10)

	go wsClient.Start()

	map1 := make(map[int]gp.ReqMessage)
	for i := 0; i < 100; i++ {
		positivePrompt := "a picture of a cat" + strconv.Itoa(i)
		msg := gp.ReqMessage{
			TaskType:       "imageInference",
			TaskUUID:       gp.GenerateUUID(),
			OutputType:     []string{"URL"},
			PositivePrompt: positivePrompt,
			Height:         512,
			Width:          512,
			Model:          "runware:100@1@1",
			Steps:          30,
			NumberResults:  1,
		}
		map1[i] = msg
	}
	go func() {
		counter := 0
		for i := 0; i < len(map1); i++ {
			select {
			case resp := <-wsClient.ReceiveMsgChan:
				log.Println(i, string(resp))
				counter++
			case err := <-wsClient.ErrChan:
				log.Println(err)
			}
		}
		log.Println(counter)
	}()

	for _, msg := range map1 {
		wsClient.SendMsgChan <- msg
		time.Sleep(1 * time.Second)
	}

}
