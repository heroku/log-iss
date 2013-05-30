package main

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

func main() {
	forwarder := NewForwarder()
	forwarder.Start()

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST is accepted", 400)
			return
		}
		if r.Header.Get("Content-Type") != "application/logplex-1" {
			http.Error(w, "Only Content-Type application/logplex-1 is accepted", 400)
			return
		}

		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, "Invalid Request", 400)
		}

		forwarder.Receive(b)
	})

	if e := http.ListenAndServe(":5000", nil); e != nil {
		log.Fatal("Unable to start HTTP server.")
	}
}

type Forwarder struct {
	Inbox   chan []byte
	c       net.Conn
	written uint64
}

func NewForwarder() *Forwarder {
	forwarder := new(Forwarder)
	forwarder.Inbox = make(chan []byte, 25*1024*1024)
	return forwarder
}

func (f *Forwarder) Receive(b []byte) {
	f.Inbox <- b
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
		log.Printf("ns=forwarder fn=periodic_stats at=emit connected=%s written=%d\n", connected, f.written)
	}
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)

	for {
		log.Println("ns=forwarder fn=connect at=start")
		c, err := net.Dial("tcp", "localhost:5001")
		if err != nil {
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
		n, err := f.c.Write(b)
		if err != nil {
			log.Println("ns=forwarder fn=write at=error message=%q\n", err)
			f.disconnect()
		} else {
			f.written += uint64(n)
			break
		}
	}
}
