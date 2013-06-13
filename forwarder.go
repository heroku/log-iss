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
	forwarder.Inbox = make(chan Message, 1024)
	forwarder.Config = config
	forwarder.Metrics = metrics
	return forwarder
}

func (f *Forwarder) Start() {
	go f.Run()
}

func (f *Forwarder) Run() {
	for m := range f.Inbox {
		f.write(m)
	}
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)
	for {
		Logf("measure.forwarder.connect.attempt=1")
		if c, err := net.Dial("tcp", f.Config.ForwardDest); err != nil {
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
			f.Metrics.Count("forwarder.write.messages")
			f.Metrics.Sum("forwarder.write.bytes", uint64(n))
			break
		}
	}
}
