package main

import (
	"log"
	"os"

	tg "github.com/MTUCI-Pixel-Team/Picture_Generator/tgBot"

	"github.com/joho/godotenv"
)

func main() {
	file, err := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Не удалось открыть файл для логов: %v", err)
	}
	defer file.Close()

	// Настраиваем логгер на запись в файл
	log.SetOutput(file)
	err = godotenv.Load()
	if err != nil {
		log.Panic(err)
	}

	// c := gp.NewWSClient(os.Getenv("API_KEY2"), 12312)

	// context, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	// defer cancel()

	// go c.Start(context)

	// for i := 0; i < 500; i++ {
	// 	promt := "cat" + strconv.Itoa(i)
	// 	msg := gp.ReqMessage{
	// 		PositivePrompt: promt,
	// 		Model:          "runware:100@1@1",
	// 		Steps:          12,
	// 		Width:          512,
	// 		Height:         512,
	// 		NumberResults:  1,
	// 		OutputType:     []string{"URL"},
	// 		TaskType:       "imageInference",
	// 		TaskUUID:       gp.GenerateUUID(),
	// 	}
	// 	fmt.Println("Sending message", i)
	// 	c.SendMsg(msg, context)
	// 	fmt.Println("Message sent", i)
	// 	resp, ok := <-c.ReceiveMsgChan
	// 	if !ok {
	// 		log.Println("Error occurred while generating a picture. Please try again or change your settings.")
	// 		continue
	// 	}
	// 	fmt.Println(i, string(resp))
	// 	if i%10 == 0 {
	// 		time.Sleep(15 * time.Second)
	// 	}
	// 	time.Sleep(1)
	// }

	bot, err := tg.NewBot(os.Getenv("TG_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	go bot.Start()

	select {}
}
