package main

import (
	"log"
	"net"
	"time"
)

type Forwarder struct {
	Inbox   chan []byte
	Dest    string
	c       net.Conn
	written uint64
}

func NewForwarder(dest string) *Forwarder {
	forwarder := new(Forwarder)
	forwarder.Inbox = make(chan []byte, 1024)
	forwarder.Dest = dest
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
		log.Printf("ns=forwarder fn=periodic_stats at=emit connected=%s written=%d inbox_count=%d\n", connected, f.written, len(f.Inbox))
	}
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)

	for {
		log.Println("ns=forwarder fn=connect at=start")
		if c, err := net.Dial("tcp", f.Dest); err != nil {
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
			f.written += uint64(n)
			break
		}
	}
}
