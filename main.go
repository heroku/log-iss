package main

import (
	"log"
	"os"
)

func main() {
	forwardDest := os.Getenv("FORWARD_DEST")
	if forwardDest == "" {
		log.Fatalln("ENV[FORWARD_DEST] is required")
	}
	forwarder := NewForwarder(forwardDest)
	forwarder.Start()

	port := os.Getenv("PORT")
	if port == "" {
		log.Fatalln("ENV[PORT] is required")
	}
	tokenMap := os.Getenv("TOKEN_MAP")
	if tokenMap == "" {
		log.Fatalln("ENV[TOKEN_MAP] is required")
	}
	tokens, err := ParseTokenMap(tokenMap)
	if err != nil {
		log.Fatalln("Unable to parse tokens:", err)
	}
	StartHttp(port, tokens, forwarder.Inbox)
}
