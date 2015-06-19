package main

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/heroku/authenticater"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/heroku/slog"
)

type Payload struct {
	SourceAddr string
	RequestId  string
	Body       []byte
	WaitCh     chan bool
}

type FixerFunc func(io.Reader, slog.Context, string, string) ([]byte, error)

type HttpServer struct {
	Config         IssConfig
	FixerFunc      FixerFunc
	Outlet         chan *Payload
	InFlightWg     sync.WaitGroup
	ShutdownCh     ShutdownCh
	isShuttingDown bool
	auth           authenticater.Authenticater
}

func NewHttpServer(config IssConfig, auth authenticater.Authenticater, fixerFunc FixerFunc, outlet chan *Payload) *HttpServer {
	return &HttpServer{
		auth:           auth,
		Config:         config,
		FixerFunc:      fixerFunc,
		Outlet:         outlet,
		ShutdownCh:     make(chan struct{}),
		isShuttingDown: false,
	}
}

func handleHTTPError(ctx slog.Context, w http.ResponseWriter, errMsg string, errCode int) {
	ctx.Count("log-iss.http.logs.post.error", 1)
	ctx.Add("post.error", errMsg)
	ctx.Add("post.code", errCode)
	http.Error(w, errMsg, errCode)
}

func (s *HttpServer) Run() error {
	go s.awaitShutdown()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx := slog.Context{}
		defer func() { LogContext(ctx) }()

		if s.isShuttingDown {
			http.Error(w, "Shutting down", 503)
			return
		}

		// check outlet depth?
		ctx.Count("log-iss.http.health.get", 1)
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		ctx := slog.Context{}
		defer func() { LogContext(ctx) }()
		defer ctx.MeasureSince("log-iss.http.logs.post.duration", time.Now())

		if s.Config.EnforceSsl && r.Header.Get("X-Forwarded-Proto") != "https" {
			handleHTTPError(ctx, w, "Only SSL requests accepted", 400)
			return
		}

		if s.isShuttingDown {
			handleHTTPError(ctx, w, "Shutting down", 503)
			return
		}

		if r.Method != "POST" {
			handleHTTPError(ctx, w, "Only POST is accepted", 400)
			return
		}

		if r.Header.Get("Content-Type") != "application/logplex-1" {
			handleHTTPError(ctx, w, "Only Content-Type application/logplex-1 is accepted", 400)
			return
		}

		if !s.auth.Authenticate(r) {
			handleHTTPError(ctx, w, "Unable to authenticate request", 401)
			return
		}

		remoteAddr := r.Header.Get("X-Forwarded-For")
		if remoteAddr == "" {
			remoteAddrParts := strings.Split(r.RemoteAddr, ":")
			remoteAddr = strings.Join(remoteAddrParts[:len(remoteAddrParts)-1], ":")
		}
		ctx.Add("remote_addr", remoteAddr)

		requestId := r.Header.Get("X-Request-Id")
		ctx.Add("request_id", requestId)

		logplexDrainToken := r.Header.Get("Logplex-Drain-Token")
		ctx.Add("logdrain_token", logplexDrainToken)

		if err, status := s.process(r.Body, ctx, remoteAddr, requestId, logplexDrainToken); err != nil {
			handleHTTPError(ctx, w, err.Error(), status)
			return
		}

		ctx.Count("log-iss.http.logs.post.success", 1)
	})

	if err := http.ListenAndServe(":"+s.Config.HttpPort, nil); err != nil {
		return err
	}

	return nil
}

func (s *HttpServer) awaitShutdown() {
	<-s.ShutdownCh
	Logf("ns=http at=shutdown")
	s.isShuttingDown = true
}

func (s *HttpServer) process(r io.Reader, ctx slog.Context, remoteAddr string, requestId string, logplexDrainToken string) (error, int) {
	s.InFlightWg.Add(1)
	defer s.InFlightWg.Done()

	var start time.Time

	fixedBody, err := s.FixerFunc(r, ctx, remoteAddr, logplexDrainToken)
	if err != nil {
		return errors.New("Problem processing body"), 400
	}

	waitCh := make(chan bool, 1)
	deadlineCh := time.After(time.Duration(5) * time.Second)

	start = time.Now()
	select {
	case s.Outlet <- &Payload{remoteAddr, requestId, fixedBody, waitCh}:
	case <-deadlineCh:
		ctx.Count("log-iss.http.logs.send.error", 1)
		return errors.New("Timeout delivering message"), 504
	}
	ctx.MeasureSince("log-iss.http.logs.send.duration", start)

	start = time.Now()
	select {
	case <-waitCh:
	case <-deadlineCh:
		ctx.Count("log-iss.http.logs.wait.error", 1)
		return errors.New("Timeout delivering message"), 504
	}
	ctx.MeasureSince("log-iss.http.logs.wait.duration", start)

	return nil, 200
}
