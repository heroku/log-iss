package main

import (
	"fmt"
	"log"
)

var Config *IssConfig

func Logf(format string, a ...interface{}) {
	orig := fmt.Sprintf(format, a...)
	fmt.Printf("app=log-iss source=%s %s\n", Config.Deploy, orig)
}

func main() {
	config, err := NewIssConfig()
	if err != nil {
		log.Fatalln(err)
	}
	Config = config

	metrics := NewMetrics()
	metrics.Start()

	forwarder := NewForwarder(Config, metrics)
	forwarder.Start()

	fixer := NewFixer(Config, forwarder.Inbox)
	fixer.Start()

	httpServer := NewHttpServer(Config, fixer.Inbox, metrics)
	err = httpServer.Run()
	if err != nil {
		log.Fatalln("Unable to start HTTP server:", err)
	}
}
