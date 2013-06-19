package main

import (
	"net"
	"time"
)

type Forwarder struct {
	Config          *IssConfig
	Metrics         *Metrics
	Inbox           chan Message
	c               net.Conn
	messagesWritten uint64
	bytesWritten    uint64
}

func NewForwarder(config *IssConfig, metrics *Metrics) *Forwarder {
	forwarder := new(Forwarder)
	forwarder.Inbox = make(chan Message)
	forwarder.Config = config
	forwarder.Metrics = metrics
	return forwarder
}

func (f *Forwarder) Start() {
	go f.Run()
}

func (f *Forwarder) Run() {
	f.Metrics.RegisterFunc(f.InboxMetrics)

	for m := range f.Inbox {
		f.write(m.Body)
		m.WaitCh <- true
	}
}

func (f *Forwarder) InboxMetrics() Measurement {
	return NewCount("forwarder.inbox.depth", uint64(len(f.Inbox)))
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)
	for {
		Logf("measure.forwarder.connect.attempt=1")
		if c, err := net.DialTimeout("tcp", f.Config.ForwardDest, f.Config.ForwardDestConnectTimeout); err != nil {
			Logf("measure.forwarder.connect.error=1 message=%q", err)
			f.disconnect()
		} else {
			Logf("measure.forwarder.connect.success=1")
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
}

func (f *Forwarder) write(b []byte) {
	for {
		f.connect()
		if n, err := f.c.Write(b); err != nil {
			Logf("measure.forwarder.write.error=1 message=%q", err)
			f.disconnect()
		} else {
			f.Metrics.Inbox <- NewCount("forwarder.write.messages", 1)
			f.Metrics.Inbox <- NewCount("forwarder.write.bytes", uint64(n))
			break
		}
	}
}
