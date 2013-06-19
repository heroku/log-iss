package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var Config *IssConfig

func Logf(format string, a ...interface{}) {
	orig := fmt.Sprintf(format, a...)
	fmt.Printf("app=log-iss source=%s %s\n", Config.Deploy, orig)
}

func awaitSigterm(ch chan int) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM)
	<-sigCh
	Logf("ns=main at=sigterm")
	ch <- 1
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
	go awaitSigterm(httpServer.ShutdownCh)
	err = httpServer.Run()
	if err != nil {
		log.Fatalln("Unable to start HTTP server:", err)
	}
}
