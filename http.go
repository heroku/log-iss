package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Payload struct {
	SourceAddr string
	RequestId  string
	Body       []byte
	WaitCh     chan bool
}

type FixerFunc func(io.Reader, string, string) ([]byte, error)

type HttpServer struct {
	Config         *IssConfig
	FixerFunc      FixerFunc
	Outlet         chan *Payload
	InFlightWg     sync.WaitGroup
	ShutdownCh     ShutdownCh
	isShuttingDown bool
}

func NewHttpServer(config *IssConfig, fixerFunc FixerFunc, outlet chan *Payload) *HttpServer {
	return &HttpServer{
		Config:         config,
		FixerFunc:      fixerFunc,
		Outlet:         outlet,
		ShutdownCh:     make(chan int),
		isShuttingDown: false,
	}
}

func (s *HttpServer) Run() error {
	go s.awaitShutdown()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if s.isShuttingDown {
			http.Error(w, "Shutting down", 503)
			return
		}

		// check outlet depth?
		Logf("measure.log-iss.http.health.get=1")
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// We either don't need the body or read it fully below, don't bother
		// trying to do anything more with it after this func returns.
		r.Header.Add("Connection", "close")

		if s.Config.EnforceSsl && r.Header.Get("X-Forwarded-Proto") != "https" {
			http.Error(w, "Only SSL requests accepted", 400)
			return
		}

		if s.isShuttingDown {
			http.Error(w, "Shutting down", 503)
			return
		}

		if r.Method != "POST" {
			http.Error(w, "Only POST is accepted", 400)
			return
		}

		if r.Header.Get("Content-Type") != "application/logplex-1" {
			http.Error(w, "Only Content-Type application/logplex-1 is accepted", 400)
			return
		}

		if err := s.checkAuth(r); err != nil {
			http.Error(w, err.Error(), 401)
			return
		}

		remoteAddr := r.Header.Get("X-Forwarded-For")
		if remoteAddr == "" {
			remoteAddrParts := strings.Split(r.RemoteAddr, ":")
			remoteAddr = strings.Join(remoteAddrParts[:len(remoteAddrParts)-1], ":")
		}

		requestId := r.Header.Get("Heroku-Request-Id")

		defer func() {
			Logf("measure.log-iss.http.logs.post.duration=%dms request_id=%q", time.Since(start)/time.Millisecond, requestId)
		}()

		if err, status := s.process(r.Body, remoteAddr, requestId); err != nil {
			http.Error(w, err.Error(), status)
			Logf("measure.log-iss.http.logs.post.error=1 message=%q request_id=%q status=%d", err, requestId, status)
			return
		}

		Logf("measure.log-iss.http.logs.post.success=1 request_id=%q", requestId)
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

func (s *HttpServer) checkAuth(r *http.Request) error {
	header := r.Header.Get("Authorization")
	if header == "" {
		return errors.New("Authorization required")
	}
	headerParts := strings.SplitN(header, " ", 2)
	if len(headerParts) != 2 {
		return errors.New("Authorization header is malformed")
	}

	method := headerParts[0]
	if method != "Basic" {
		return errors.New("Only Basic Authorization is accepted")
	}

	encodedUserPass := headerParts[1]
	decodedUserPass, err := base64.StdEncoding.DecodeString(encodedUserPass)
	if err != nil {
		return errors.New("Authorization header is malformed")
	}

	userPassParts := bytes.SplitN(decodedUserPass, []byte{':'}, 2)
	if len(userPassParts) != 2 {
		return errors.New("Authorization header is malformed")
	}

	user := userPassParts[0]
	pass := userPassParts[1]
	token, ok := s.Config.Tokens[string(user)]
	if !ok {
		return errors.New("Unknown user")
	}
	if token != string(pass) {
		return errors.New("Incorrect token")
	}

	return nil
}

func (s *HttpServer) process(r io.Reader, remoteAddr string, requestId string) (error, int) {
	s.InFlightWg.Add(1)
	defer s.InFlightWg.Done()

	var start time.Time

	start = time.Now()
	fixedBody, err := s.FixerFunc(r, remoteAddr, requestId)
	if err != nil {
		return errors.New("Problem processing body"), 400
	}
	Logf("measure.log-iss.http.logs.fixer-func.duration=%dms request_id=%q", time.Since(start)/time.Millisecond, requestId)

	waitCh := make(chan bool)
	deadlineCh := time.After(time.Duration(5) * time.Second)

	start = time.Now()
	select {
	case s.Outlet <- &Payload{remoteAddr, requestId, fixedBody, waitCh}:
	case <-deadlineCh:
		return errors.New("Problem delivering message"), 504
	}
	Logf("measure.log-iss.http.logs.send.duration=%dms request_id=%q", time.Since(start)/time.Millisecond, requestId)

	start = time.Now()
	select {
	case <-waitCh:
	case <-deadlineCh:
		return errors.New("Problem delivering message"), 504
	}
	Logf("measure.log-iss.http.logs.wait.duration=%dms request_id=%q", time.Since(start)/time.Millisecond, requestId)

	return nil, 200
}
