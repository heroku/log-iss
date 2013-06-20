package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

type Payload struct {
	SourceAddr string
	Body       []byte
	WaitCh     chan bool
}

type HttpServer struct {
	Config         *IssConfig
	Metrics        *Metrics
	Outlet         chan Payload
	ShutdownCh     chan int
	isShuttingDown bool
}

func NewHttpServer(config *IssConfig, outlet chan Payload, metrics *Metrics) *HttpServer {
	return &HttpServer{config, metrics, outlet, make(chan int), false}
}

func (s *HttpServer) Run() error {
	go s.awaitShutdown()

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if s.isShuttingDown {
			http.Error(w, "Shutting down", 503)
			return
		}

		// check outlet depth?
		Logf("measure.http.health.get=1")
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
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

		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, "Invalid Request", 400)
			return
		}

		remoteAddr := r.Header.Get("X-Forwarded-For")
		if remoteAddr == "" {
			remoteAddrParts := strings.Split(r.RemoteAddr, ":")
			remoteAddr = strings.Join(remoteAddrParts[:len(remoteAddrParts)-1], ":")
		}

		s.Metrics.Inbox <- NewCount("http.logs.post", 1)
		s.sendAndWait(remoteAddr, b)
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

func (s *HttpServer) sendAndWait(remoteAddr string, b []byte) {
	waitCh := make(chan bool)
	s.Outlet <- Payload{remoteAddr, b, waitCh}
	<-waitCh
}
