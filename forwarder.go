package main

import (
	"log"
	"net"
	"time"
)

type Forwarder struct {
	Config          *Config
	Inbox           chan Message
	c               net.Conn
	messagesWritten uint64
	bytesWritten    uint64
}

func NewForwarder(c *Config) *Forwarder {
	forwarder := new(Forwarder)
	forwarder.Inbox = make(chan Message, 1024)
	forwarder.Config = c
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
	var connected string
	t := time.Tick(1 * time.Second)

	for {
		<-t
		connected = "no"
		if f.c != nil {
			connected = "yes"
		}
		log.Printf("ns=forwarder fn=periodic_stats at=emit connected=%s messages_written=%d bytes_written=%d inbox_count=%d\n", connected, f.messagesWritten, f.bytesWritten, len(f.Inbox))
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
			log.Printf("ns=forwarder fn=connect at=error message=%q\n", err)
			f.disconnect()
		} else {
			log.Println("ns=forwarder fn=connect at=finish")
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
			log.Printf("ns=forwarder fn=write at=error message=%q\n", err)
			f.disconnect()
		} else {
			f.messagesWritten += 1
			f.bytesWritten += uint64(n)
			break
		}
	}
}
