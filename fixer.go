package main

import (
	"bytes"
	"github.com/bmizerany/lpx"
	"io/ioutil"
	"strconv"
)

type MessageBody []byte

type Message struct {
	Body   MessageBody
	WaitCh chan bool
}

type Fixer struct {
	Config *IssConfig
	Inbox  chan Payload
	Outlet chan Message
}

func NewFixer(config *IssConfig, outlet chan Message) *Fixer {
	return &Fixer{config, make(chan Payload), outlet}
}

func (f *Fixer) Start() {
	go f.Run()
}

func (f *Fixer) Run() {
	for p := range f.Inbox {
		for _, fixed := range Fix(p) {
			f.sendAndWait(fixed)
		}
		p.WaitCh <- true
	}
}

func (f *Fixer) sendAndWait(messageBody MessageBody) {
	waitCh := make(chan bool)
	f.Outlet <- Message{messageBody, waitCh}
	<-waitCh
}

func Fix(payload Payload) []MessageBody {
	nilVal := []byte(`- `)

	messages := make([]MessageBody, 0)

	lp := lpx.NewReader(bytes.NewBuffer(payload.Body))
	for lp.Next() {
		header := lp.Header()

		// LEN SP PRI VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID SP STRUCTURED-DATA MSG
		var messageWriter bytes.Buffer
		messageWriter.Write(header.PrivalVersion)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Time)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Hostname)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Name)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Procid)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Msgid)
		messageWriter.WriteString(" [origin ip=\"")
		messageWriter.WriteString(payload.SourceAddr)
		messageWriter.WriteString("\"]")

		b := lp.Bytes()
		if bytes.Equal(b[0:2], nilVal) {
			messageWriter.Write(b[1:])
		} else {
			if b[0] != '[' {
				messageWriter.WriteString(" ")
			}
			messageWriter.Write(b)
		}

		var messageLenWriter bytes.Buffer
		messageLenWriter.WriteString(strconv.Itoa(messageWriter.Len()))
		messageLenWriter.WriteString(" ")
		messageWriter.WriteTo(&messageLenWriter)

		if fullMessage, err := ioutil.ReadAll(&messageLenWriter); err != nil {
			Logf("measure.fixer.fix.error.readall=1 message=%q", err)
			continue
		} else {
			messages = append(messages, fullMessage)
		}
	}

	if lp.Err() != nil {
		Logf("measure.fixer.fix.error.lpx=1 message=%q", lp.Err())
	}

	return messages
}
