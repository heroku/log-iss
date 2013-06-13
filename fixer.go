package main

import (
	"bytes"
	"fmt"
	"github.com/bmizerany/lpx"
	"io/ioutil"
	"strconv"
)

type Message []byte

type Fixer struct {
	Config *Config
	Inbox  chan Payload
	Outlet chan Message
}

func NewFixer(config *Config, outlet chan Message) *Fixer {
	return &Fixer{config, make(chan Payload), outlet}
}

func (f *Fixer) Start() {
	go f.Run()
}

func (f *Fixer) Run() {
	for p := range f.Inbox {
		for _, fixed := range Fix(p) {
			f.Outlet <- fixed
		}
	}
}

func Fix(payload Payload) []Message {
	nilVal := []byte(`- `)

	messages := make([]Message, 0)

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
			fmt.Println("error reading full message:", err)
			continue
		} else {
			messages = append(messages, fullMessage)
		}
	}

	if lp.Err() != nil {
		fmt.Println("error from lp:", lp.Err())
	}

	return messages
}
