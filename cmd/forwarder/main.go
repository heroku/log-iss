package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rcrowley/go-metrics/librato"

	log "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/heroku/authenticater"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/heroku/rollrus"
)

type ShutdownCh chan struct{}

var Config IssConfig

func awaitShutdownSignals(chs ...ShutdownCh) {
	sigCh := make(chan os.Signal)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	for sig := range sigCh {
		log.WithFields(log.Fields{"at": "shutdown-signal", "signal": sig}).Info()
		for _, ch := range chs {
			ch <- struct{}{}
		}
	}
}

func main() {
	rollrus.SetupLogging(os.Getenv("ROLLBAR_TOKEN"), os.Getenv("ENVIRONMENT"))

	config, err := NewIssConfig()
	if err != nil {
		log.Fatalln(err)
	}

	Config = config

	log.AddHook(&DefaultFieldsHook{log.Fields{"app": "log-iss", "source": Config.Deploy}})

	auth, err := authenticater.NewBasicAuthFromString(Config.Tokens)
	if err != nil {
		log.Fatalln(err)
	}

	forwarderSet := NewForwarderSet(Config)

	shutdownCh := make(ShutdownCh)

	httpServer := NewHttpServer(Config, auth, Fix, forwarderSet)

	go awaitShutdownSignals(httpServer.ShutdownCh, shutdownCh)

	go forwarderSet.Run()

	go func() {
		if err := httpServer.Run(); err != nil {
			log.Fatalln("Unable to start HTTP server:", err)
		}
	}()

	if Config.LibratoOwner != "" && Config.LibratoToken != "" {
		log.Info("starting librato metrics reporting")
		go librato.Librato(
			config.MetricsRegistry,
			20*time.Second,
			Config.LibratoOwner,
			Config.LibratoToken,
			Config.LibratoSource,
			[]float64{0.50, 0.95, 0.99},
			time.Millisecond,
		)
	}

	log.WithField("at", "start").Info()
	<-shutdownCh
	log.WithField("at", "drain").Info()
	httpServer.Wait()
	log.WithField("at", "exit").Info()
}
