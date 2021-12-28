package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	librato "github.com/heroku/go-metrics-librato"
	"github.com/heroku/rollrus"
	log "github.com/sirupsen/logrus"
)

type shutdownCh chan struct{}

func awaitShutdownSignals(chs ...shutdownCh) {
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

	log.AddHook(&DefaultFieldsHook{log.Fields{"app": "log-iss", "source": config.Deploy}})

	authConfig, err := NewAuthConfig()
	if err != nil {
		log.Fatalln(err)
	}

	auth, err := newAuth(authConfig, config.MetricsRegistry)
	if err != nil {
		log.Fatalln(err)
	}

	forwarderSet := newForwarderSet(config)

	shutdownCh := make(shutdownCh)
	httpServer := newHTTPServer(config, auth, fix, forwarderSet)

	go awaitShutdownSignals(httpServer.shutdownCh, shutdownCh)

	go forwarderSet.Run()

	go func() {
		if err := httpServer.Run(); err != nil {
			log.Fatalln("Unable to start HTTP server:", err)
		}
	}()

	if config.LibratoOwner != "" && config.LibratoToken != "" {
		log.WithField("source", config.LibratoSource).Info("starting librato metrics reporting")
		go librato.Librato(
			context.Background(),
			config.MetricsRegistry,
			20*time.Second,
			config.LibratoOwner,
			config.LibratoToken,
			"",
			config.LibratoSource,
			[]float64{0.50, 0.95, 0.99},
			time.Millisecond,
			// Counters are Gauges now - we need heroku/go-metrics-librato to reset gauges upon submission
			// so they don't constantly build up
			true,
		)
	}

	log.WithField("at", "start").Info()
	<-shutdownCh
	log.WithField("at", "drain").Info()
	httpServer.Wait()
	log.WithField("at", "exit").Info()
}
