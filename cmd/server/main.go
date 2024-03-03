package main

import (
	"log"
	"os"

	"github.com/cameroncuttingedge/terminal-chat/chat"
	"github.com/cameroncuttingedge/terminal-chat/security"
)

func main() {
	chat.StartServer()
}

func init() {
	if err := security.GenerateSelfSignedCertificate(); err != nil {
		log.Fatalf("Failed to generate self-signed certificate: %v", err)
	}
	if os.Getenv("LOGGING") == "1" {
		logFile, err := os.OpenFile("chat_server.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			log.Fatalln("Failed to open log file:", err)
		}
		log.SetOutput(logFile)
	} else {
		// Correctly discard all log output by opening /dev/null as a file
		devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0666)
		if err != nil {
			log.Fatalln("Failed to discard log output:", err)
		}
		log.SetOutput(devNull)
	}
}
