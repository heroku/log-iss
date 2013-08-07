package main

import (
	"net"
	"time"
)

type ForwarderSet struct {
	Config *IssConfig
	Inbox  chan *Payload
}

type Forwarder struct {
	Id  int
	Set *ForwarderSet
	c   net.Conn
}

func NewForwarderSet(config *IssConfig) *ForwarderSet {
	return &ForwarderSet{
		Config: config,
		Inbox:  make(chan *Payload, 1000),
	}
}

func (fs *ForwarderSet) Start() {
	for i := 0; i < 4; i++ {
		forwarder := NewForwarder(fs, i)
		forwarder.Start()
	}
}

func NewForwarder(set *ForwarderSet, id int) *Forwarder {
	return &Forwarder{
		Id:  id,
		Set: set,
	}
}

func (f *Forwarder) Start() {
	go f.Run()
}

func (f *Forwarder) Run() {
	for p := range f.Set.Inbox {
		start := time.Now()
		f.write(p)
		p.WaitCh <- true
		Logf("measure.log-iss.forwarder.process.duration=%dms id=%d request_id=%q", time.Since(start)/time.Millisecond, f.Id, p.RequestId)
	}
}

func (f *Forwarder) connect() {
	if f.c != nil {
		return
	}

	rate := time.Tick(200 * time.Millisecond)
	for {
		start := time.Now()
		Logf("measure.log-iss.forwarder.connect.attempt=1 id=%d", f.Id)
		if c, err := net.DialTimeout("tcp", f.Set.Config.ForwardDest, f.Set.Config.ForwardDestConnectTimeout); err != nil {
			Logf("measure.log-iss.forwarder.connect.error=1 id=%d message=%q", f.Id, err)
			f.disconnect()
		} else {
			Logf("measure.log-iss.forwarder.connect.success=1 measure.log-iss.forwarder.connect.duration=%dms id=%d remote_addr=%s", time.Since(start)/time.Millisecond, f.Id, c.RemoteAddr().String())
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
	Logf("measure.log-iss.forwarder.disconnect.success=1 id=%d", f.Id)
}

func (f *Forwarder) write(p *Payload) {
	for {
		f.connect()

		f.c.SetWriteDeadline(time.Now().Add(1 * time.Second))
		if n, err := f.c.Write(p.Body); err != nil {
			Logf("measure.log-iss.forwarder.write.error=1 id=%d request_id=%q message=%q remote_addr=%s", f.Id, p.RequestId, err, f.c.RemoteAddr().String())
			f.disconnect()
		} else {
			Logf("measure.log-iss.forwarder.write.success.messages=1 measure.log-iss.forwarder.write.success.bytes=%d id=%d request_id=%q remote_addr=%s", n, f.Id, p.RequestId, f.c.RemoteAddr().String())
			return
		}
	}
}
