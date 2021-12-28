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

	"github.com/heroku/go-metrics"
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

// FixerFunc params:
//  * http.Request -- incoming http request
//  * io.Reader - request body stream
//  * string - remote address of incoming request
//  * string - logplex drain token
//  * metadataId - ID to use when adding metadata to logs
//  * credential - the credential used to authenticate
//  * []string - a slice of custom query paramters to look for in the request
// FixerFunc returns:
//  * boolean - indicating whether the request has query params (aka metadata).
//  * int64  - number of log lines read from the stream
//  * error - if something went wrong.
type FixerFunc func(*http.Request, io.Reader, string, string, string, *credential, *IssConfig) (fixResult, error)

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
	pHostnameTruncations  metrics.Counter // tracks the number of hostname fields in logs that have been truncated
	pAppnameTruncations   metrics.Counter // tracks the number of appname fields in logs that have been truncated
	pProcidTruncations    metrics.Counter // tracks the number of procid fields in logs that have been truncated
	pMsgidTruncations     metrics.Counter // trakcs the number of msgid fields in logs that have been truncated
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
		pHostnameTruncations:  metrics.GetOrRegisterCounter("log-iss.logs.hostname_truncations", config.MetricsRegistry),
		pAppnameTruncations:   metrics.GetOrRegisterCounter("log-iss.logs.appname_truncations", config.MetricsRegistry),
		pProcidTruncations:    metrics.GetOrRegisterCounter("log-iss.logs.procid_truncations", config.MetricsRegistry),
		pMsgidTruncations:     metrics.GetOrRegisterCounter("log-iss.logs.msgid_truncations", config.MetricsRegistry),
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

		cred := s.auth.Authenticate(r)
		if cred == nil {
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

		if err, status := s.process(r, body, remoteAddr, requestID, logplexDrainToken, s.Config.MetadataId, cred); err != nil {
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

func (s *httpServer) process(req *http.Request, reader io.Reader, remoteAddr string, requestID string, logplexDrainToken string, metadataId string, cred *credential) (error, int) {
	s.Add(1)
	defer s.Done()

	r, err := s.FixerFunc(req, reader, remoteAddr, logplexDrainToken, metadataId, cred, &s.Config)
	if err != nil {
		return errors.New("Problem fixing body: " + err.Error()), http.StatusBadRequest
	}

	s.pLogsReceived.Inc(r.numLogs)
	if r.hasMetadata {
		s.pMetadataLogsReceived.Inc(r.numLogs)
	}

	payload := NewPayload(remoteAddr, requestID, r.bytes)
	if err := s.deliverer.Deliver(payload); err != nil {
		return errors.New("Problem delivering body: " + err.Error()), http.StatusGatewayTimeout
	}

	s.pLogsSent.Inc(r.numLogs)
	if r.hasMetadata {
		s.pMetadataLogsSent.Inc(r.numLogs)
	}
	s.pHostnameTruncations.Inc(r.hostnameTruncs)
	s.pAppnameTruncations.Inc(r.appnameTruncs)
	s.pProcidTruncations.Inc(r.procidTruncs)
	s.pMsgidTruncations.Inc(r.msgidTruncs)

	return nil, 200
}
