package main

import (
	"log"
	"os"
)

type Config struct {
	Deploy      string
	ForwardDest string
	HttpPort    string
	Tokens      Tokens
}

func main() {
	deploy := os.Getenv("DEPLOY")
	if deploy == "" {
		log.Fatalln("ENV[DEPLOY] is required")
	}
	forwardDest := os.Getenv("FORWARD_DEST")
	if forwardDest == "" {
		log.Fatalln("ENV[FORWARD_DEST] is required")
	}
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

	config := new(Config)
	config.Deploy = deploy
	config.ForwardDest = forwardDest
	config.HttpPort = port
	config.Tokens = tokens

	forwarder := NewForwarder(config)
	forwarder.Start()

	fixer := NewFixer(config, forwarder.Inbox)
	fixer.Start()

	httpServer := NewHttpServer(config, fixer.Inbox)
	err = httpServer.Run()
	if err != nil {
		log.Fatalln("Unable to start HTTP server:", err)
	}
}
