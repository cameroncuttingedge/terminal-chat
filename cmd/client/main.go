package main

import (
	"log"
	"os"

	"github.com/cameroncuttingedge/terminal-chat/chat"
)

func main() {
	chat.StartClient()
}

func init() {
	logFile, err := os.OpenFile("chat_client.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file:", err)
	}
	log.SetOutput(logFile)
}
