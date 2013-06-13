package main

import (
	"log"
	"net"
	"time"
)

type Forwarder struct {
	Config          *IssConfig
	Inbox           chan Message
	c               net.Conn
	messagesWritten uint64
	bytesWritten    uint64
}

func NewForwarder(config *IssConfig) *Forwarder {
	forwarder := new(Forwarder)
	forwarder.Inbox = make(chan Message, 1024)
	forwarder.Config = config
	return forwarder
}

func (f *Forwarder) Start() {
	go f.Run()
	go f.PeriodicStats()
}

func (f *Forwarder) Run() {
	for m := range f.Inbox {
		f.write(m)
	}
}

func (f *Forwarder) PeriodicStats() {
	var connected int
	t := time.Tick(1 * time.Second)

	for {
		<-t
		connected = 0
		if f.c != nil {
			connected = 1
		}
		Logf("measure.forwarder.messages.written=%d measure.forwarder.bytes.written=%d measure.forwarder.inbox.depth=%d measure.forwarder.connected=%d\n",
			f.messagesWritten, f.bytesWritten, len(f.Inbox), connected)
	}
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)

	for {
		log.Println("ns=forwarder fn=connect at=start")
		if c, err := net.Dial("tcp", f.Config.ForwardDest); err != nil {
			Logf("measure.forwarder.connect.error message=%q\n", err)
			f.disconnect()
		} else {
			Logf("measure.forwarder.connect.success\n")
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
			Logf("measure.forwarder.write.error message=%q\n", err)
			f.disconnect()
		} else {
			f.messagesWritten += 1
			f.bytesWritten += uint64(n)
			break
		}
	}
}
