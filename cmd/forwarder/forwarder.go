package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/heroku/go-metrics"
	log "github.com/sirupsen/logrus"
)

type deliverer interface {
	Deliver(p payload) error
}

type forwarderSet struct {
	Config  IssConfig
	Inbox   chan payload
	timeout metrics.Counter // counts how many times we times out waiting for delivery notification
	full    metrics.Counter // counts how many times the queue was full
}

func newForwarderSet(config IssConfig) *forwarderSet {
	return &forwarderSet{
		Config:  config,
		Inbox:   make(chan payload, 1000),
		timeout: metrics.GetOrRegisterCounter("log-iss.forwardset.deliver.timeout", config.MetricsRegistry),
		full:    metrics.GetOrRegisterCounter("log-iss.forwardset.deliver.full", config.MetricsRegistry),
	}
}

func (fs *forwarderSet) Run() {
	for i := 0; i < fs.Config.ForwardCount; i++ {
		forwarder := newForwarder(fs.Config, fs.Inbox, i)
		go forwarder.Run()
	}
}

func (fs *forwarderSet) Deliver(p payload) (err error) {
	deadline := time.After(time.Second * 5)

	select {
	case fs.Inbox <- p:
	case <-deadline:
		fs.full.Inc(1)
		return fmt.Errorf("ForwardSet queue full too long.")
	}

	select {
	case <-p.WaitCh:
		// FIXME: delivery duration?
	case <-deadline:
		fs.timeout.Inc(1)
		return fmt.Errorf("Timed out awaiting delivery notification for payload")
	}

	return nil
}

type forwarder struct {
	ID           int
	Config       IssConfig
	Inbox        chan payload
	c            net.Conn
	duration     metrics.Timer   // tracks how long it takes to forward messages
	cDisconnects metrics.Counter // counts disconnects
	cSuccesses   metrics.Counter // counts connection successes
	cErrors      metrics.Counter // counts connection errors
	wErrors      metrics.Counter // counts write errors
	wSuccesses   metrics.Counter // counts write successes
	wBytes       metrics.Counter // counts written bytes
}

func newForwarder(config IssConfig, inbox chan payload, id int) *forwarder {
	me := fmt.Sprintf("log-iss.forwarder.%d", id)
	return &forwarder{
		ID:           id,
		Config:       config,
		Inbox:        inbox,
		duration:     metrics.GetOrRegisterTimer(me+".duration", config.MetricsRegistry),
		cDisconnects: metrics.GetOrRegisterCounter(me+".disconnects", config.MetricsRegistry),
		cSuccesses:   metrics.GetOrRegisterCounter(me+".connect.successes", config.MetricsRegistry),
		cErrors:      metrics.GetOrRegisterCounter(me+".connect.errors", config.MetricsRegistry),
		wErrors:      metrics.GetOrRegisterCounter(me+".write.errors", config.MetricsRegistry),
		wSuccesses:   metrics.GetOrRegisterCounter(me+".write.successes", config.MetricsRegistry),
		wBytes:       metrics.GetOrRegisterCounter(me+".write.bytes", config.MetricsRegistry),
	}
}

func (f *forwarder) Run() {
	for p := range f.Inbox {
		start := time.Now()
		f.write(p)
		p.WaitCh <- struct{}{}
		f.duration.UpdateSince(start)
	}
}

func (f *forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)
	for {
		var c net.Conn
		var err error

		if f.Config.TlsConfig != nil {
			c, err = tls.Dial("tcp", f.Config.ForwardDest, f.Config.TlsConfig)
		} else {
			c, err = net.DialTimeout("tcp", f.Config.ForwardDest, f.Config.ForwardDestConnectTimeout)
		}

		if err != nil {
			f.cErrors.Inc(1)
			log.WithFields(log.Fields{"id": f.ID, "message": err}).Error("Forwarder Connection Error")
			f.disconnect()
		} else {
			f.cSuccesses.Inc(1)
			log.WithFields(log.Fields{"id": f.ID, "remote_addr": c.RemoteAddr().String()}).Info("Forwarder Connection Success")
			f.c = c
			return
		}
		<-rate
	}
}

func (f *forwarder) disconnect() {
	if f.c != nil {
		f.c.Close()
	}
	f.c = nil
	f.cDisconnects.Inc(1)
}

func (f *forwarder) write(p payload) {
	for {
		f.connect()

		f.c.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if n, err := f.c.Write(p.Body); err != nil {
			f.wErrors.Inc(1)
			log.WithFields(log.Fields{"id": f.ID, "request_id": p.RequestID, "err": err, "remote": f.c.RemoteAddr().String()}).Error("Error writing payload")
			f.disconnect()
		} else {
			f.wSuccesses.Inc(1)
			f.wBytes.Inc(int64(n))
			return
		}
	}
}
