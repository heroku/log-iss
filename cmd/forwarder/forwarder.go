package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	log "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	metrics "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

type Deliverer interface {
	Deliver(p Payload) error
}

type ForwarderSet struct {
	Config  IssConfig
	Inbox   chan Payload
	timeout metrics.Counter // counts how many times we times out waiting for delivery notification
	full    metrics.Counter // counts how many times the queue was full
}

func NewForwarderSet(config IssConfig) *ForwarderSet {
	return &ForwarderSet{
		Config:  config,
		Inbox:   make(chan Payload, 1000),
		timeout: metrics.GetOrRegisterCounter("forwardset.deliver.timeout", config.MetricsRegistry),
		full:    metrics.GetOrRegisterCounter("forwardset.deliver.full", config.MetricsRegistry),
	}
}

func (fs *ForwarderSet) Run() {
	for i := 0; i < fs.Config.ForwardCount; i++ {
		forwarder := NewForwarder(fs.Config, fs.Inbox, i)
		go forwarder.Run()
	}
}

func (fs *ForwarderSet) Deliver(p Payload) (err error) {
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

type Forwarder struct {
	Id           int
	Config       IssConfig
	Inbox        chan Payload
	c            net.Conn
	duration     metrics.Timer   // tracks how long it takes to forward messages
	cDisconnects metrics.Counter // counts disconnects
	cSuccesses   metrics.Counter // counts connection successes
	cErrors      metrics.Counter // counts connection errors
	wErrors      metrics.Counter // counts write errors
	wSuccesses   metrics.Counter // counts write successes
	wBytes       metrics.Counter // counts written bytes
}

func NewForwarder(config IssConfig, inbox chan Payload, id int) *Forwarder {
	me := fmt.Sprintf("forwarder.%i", id)
	return &Forwarder{
		Id:           id,
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

func (f *Forwarder) Run() {
	for p := range f.Inbox {
		start := time.Now()
		f.write(p)
		p.WaitCh <- struct{}{}
		f.duration.UpdateSince(start)
	}
}

func (f *Forwarder) connect() {
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
			log.WithFields(log.Fields{"id": f.Id, "message": err}).Error("Forwarder Connection Error")
			f.disconnect()
		} else {
			log.WithFields(log.Fields{"id": f.Id, "remote_addr": c.RemoteAddr().String()}).Info("Forwarder Connection Success")
			f.c = c
			return
		}
		<-rate
	}
}

func (f *Forwarder) disconnect() {
	if f.c != nil {
		f.c.Close()
	}
	f.c = nil
	f.cDisconnects.Inc(1)
}

func (f *Forwarder) write(p Payload) {
	for {
		f.connect()

		f.c.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if n, err := f.c.Write(p.Body); err != nil {
			f.wErrors.Inc(1)
			log.WithFields(log.Fields{"id": f.Id, "request_id": p.RequestId, "err": err, "remote": f.c.RemoteAddr().String()}).Error("Error writing payload")
			f.disconnect()
		} else {
			f.wSuccesses.Inc(1)
			f.wBytes.Inc(int64(n))
			return
		}
	}
}
