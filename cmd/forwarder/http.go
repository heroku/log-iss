package main

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type payload struct {
	SourceAddr string
	RequestID  string
	Body       []byte
	WaitCh     chan struct{}
}

func NewPayload(sa string, ri string, b []byte) payload {
	return payload{
		SourceAddr: sa,
		RequestID:  ri,
		Body:       b,
		WaitCh:     make(chan struct{}, 1),
	}
}

type FixerFunc func(*http.Request, io.Reader, string, string, string) (bool, int64, []byte, error)

type httpServer struct {
	Config                IssConfig
	FixerFunc             FixerFunc
	shutdownCh            shutdownCh
	deliverer             deliverer
	isShuttingDown        bool
	auth                  *BasicAuth
	posts                 metrics.Timer   // tracks metrics about posts
	healthChecks          metrics.Timer   // tracks metrics about health checks
	pErrors               metrics.Counter // tracks the count of post errors
	pSuccesses            metrics.Counter // tracks the number of post successes
	pAuthErrors           metrics.Counter // tracks the count of auth errors
	pAuthSuccesses        metrics.Counter // tracks the number of auth successes
	pMetadataLogsReceived metrics.Counter // tracks the number of logs that have metadata that have been received
	pLogsReceived         metrics.Counter // tracks the number of logs that have been received
	pMetadataLogsSent     metrics.Counter // tracks the number of logs that have metadata that have been received
	pLogsSent             metrics.Counter // tracks the number of logs that have been received
	pAuthUsers            map[string]metrics.Counter
	sync.WaitGroup
}

func newHTTPServer(config IssConfig, auth *BasicAuth, fixerFunc FixerFunc, deliverer deliverer) *httpServer {
	return &httpServer{
		auth:                  auth,
		Config:                config,
		FixerFunc:             fixerFunc,
		deliverer:             deliverer,
		shutdownCh:            make(shutdownCh),
		posts:                 metrics.GetOrRegisterTimer("log-iss.http.logs", config.MetricsRegistry),
		healthChecks:          metrics.GetOrRegisterTimer("log-iss.http.healthchecks", config.MetricsRegistry),
		pErrors:               metrics.GetOrRegisterCounter("log-iss.http.logs.errors", config.MetricsRegistry),
		pSuccesses:            metrics.GetOrRegisterCounter("log-iss.http.logs.successes", config.MetricsRegistry),
		pAuthErrors:           metrics.GetOrRegisterCounter("log-iss.auth.errors", config.MetricsRegistry),
		pAuthSuccesses:        metrics.GetOrRegisterCounter("log-iss.auth.successes", config.MetricsRegistry),
		pMetadataLogsReceived: metrics.GetOrRegisterCounter("log-iss.metadata_logs.received", config.MetricsRegistry),
		pLogsReceived:         metrics.GetOrRegisterCounter("log-iss.logs.received", config.MetricsRegistry),
		pMetadataLogsSent:     metrics.GetOrRegisterCounter("log-iss.metadata_logs.sent", config.MetricsRegistry),
		pLogsSent:             metrics.GetOrRegisterCounter("log-iss.logs.sent", config.MetricsRegistry),
		pAuthUsers:            make(map[string]metrics.Counter),
		isShuttingDown:        false,
	}
}

func (s *httpServer) handleHTTPError(w http.ResponseWriter, errMsg string, errCode int, fields ...log.Fields) {
	ff := log.Fields{"post.code": errCode}
	for _, f := range fields {
		for k, v := range f {
			ff[k] = v
		}
	}

	s.pErrors.Inc(1)
	log.WithFields(ff).Error(errMsg)
	http.Error(w, errMsg, errCode)
}

func extractRemoteAddr(r *http.Request) string {
	remoteAddr := r.Header.Get("X-Forwarded-For")
	if remoteAddr == "" {
		remoteAddrParts := strings.Split(r.RemoteAddr, ":")
		remoteAddr = strings.Join(remoteAddrParts[:len(remoteAddrParts)-1], ":")
	}
	return remoteAddr
}

func (s *httpServer) Run() error {
	go s.awaitShutdown()

	//FXME: check outlet depth?
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		defer s.healthChecks.UpdateSince(time.Now())
		if s.isShuttingDown {
			http.Error(w, "Shutting down", 503)
			return
		}

	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		defer s.posts.UpdateSince(time.Now())

		if s.Config.EnforceSsl && r.Header.Get("X-Forwarded-Proto") != "https" {
			s.handleHTTPError(w, "Only SSL requests accepted", 400)
			return
		}

		if s.isShuttingDown {
			s.handleHTTPError(w, "Shutting down", 503)
			return
		}

		if r.Method != "POST" {
			s.handleHTTPError(w, "Only POST is accepted", 400)
			return
		}

		if r.Header.Get("Content-Type") != "application/logplex-1" {
			s.handleHTTPError(w, "Only Content-Type application/logplex-1 is accepted", 400)
			return
		}

		if !s.auth.Authenticate(r) {
			s.pAuthErrors.Inc(1)
			s.handleHTTPError(w, "Unable to authenticate request", 401)
			return
		} else {
			s.pAuthSuccesses.Inc(1)
		}

		remoteAddr := extractRemoteAddr(r)
		requestID := r.Header.Get("X-Request-Id")
		logplexDrainToken := r.Header.Get("Logplex-Drain-Token")

		body := r.Body
		var err error

		if r.Header.Get("Content-Encoding") == "gzip" {
			body, err = gzip.NewReader(r.Body)
			if err != nil {
				s.handleHTTPError(w, "Could not decode gzip request", 500)
				return
			}
			defer body.Close()
		}

		// This should only be reached if authentication information is valid.
		if authUser, _, ok := r.BasicAuth(); ok {
			var um metrics.Counter
			um, ok = s.pAuthUsers[authUser]
			if !ok {
				if s.Config.Debug {
					fmt.Printf("DEBUG: create: log-iss.auth.user.%s\n", authUser)
				}
				um = metrics.GetOrRegisterCounter(fmt.Sprintf("log-iss.auth.user.%s", authUser), s.Config.MetricsRegistry)
				s.pAuthUsers[authUser] = um
			}

			if s.Config.Debug {
				fmt.Printf("DEBUG: log-iss.auth.user.%s++\n", authUser)
			}
			um.Inc(1)
		}

		if err, status := s.process(r, body, remoteAddr, requestID, logplexDrainToken, s.Config.MetadataId); err != nil {
			s.handleHTTPError(
				w, err.Error(), status,
				log.Fields{"remote_addr": remoteAddr, "requestId": requestID, "logdrain_token": logplexDrainToken},
			)
			return
		}

		s.pSuccesses.Inc(1)
	})

	return http.ListenAndServe(":"+s.Config.HttpPort, nil)
}

func (s *httpServer) awaitShutdown() {
	<-s.shutdownCh
	s.isShuttingDown = true
	log.WithFields(log.Fields{"ns": "http", "at": "shutdown"}).Info()
}

func (s *httpServer) process(req *http.Request, r io.Reader, remoteAddr string, requestID string, logplexDrainToken string, metadataId string) (error, int) {
	s.Add(1)
	defer s.Done()

	hasMetadata, numLogs, fixedBody, err := s.FixerFunc(req, r, remoteAddr, logplexDrainToken, metadataId)
	if err != nil {
		return errors.New("Problem fixing body: " + err.Error()), http.StatusBadRequest
	}

	s.pLogsReceived.Inc(numLogs)
	if hasMetadata {
		s.pMetadataLogsReceived.Inc(numLogs)
	}

	payload := NewPayload(remoteAddr, requestID, fixedBody)
	if err := s.deliverer.Deliver(payload); err != nil {
		return errors.New("Problem delivering body: " + err.Error()), http.StatusGatewayTimeout
	}

	s.pLogsSent.Inc(numLogs)
	if hasMetadata {
		s.pMetadataLogsSent.Inc(numLogs)
	}

	return nil, 200
}
