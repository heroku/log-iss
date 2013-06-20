package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type ShutdownCh chan int

var Config *IssConfig

func Logf(format string, a ...interface{}) {
	orig := fmt.Sprintf(format, a...)
	fmt.Printf("app=log-iss source=%s %s\n", Config.Deploy, orig)
}

func awaitSigterm(chs []ShutdownCh) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM)
	<-sigCh
	Logf("ns=main at=sigterm")
	for _, ch := range chs {
		ch <- 1
	}
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

	shutdownCh := make(ShutdownCh)

	httpServer := NewHttpServer(Config, fixer.Inbox, metrics)

	go awaitSigterm([]ShutdownCh{httpServer.ShutdownCh, shutdownCh})

	go func() {
		if err := httpServer.Run(); err != nil {
			log.Fatalln("Unable to start HTTP server:", err)
		}
	}()

	<-shutdownCh
	httpServer.InFlightWg.Wait()
}
