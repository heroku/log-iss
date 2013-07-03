package main

import (
	"bufio"
	"bytes"
	"github.com/bmizerany/lpx"
	"io"
	"io/ioutil"
	"strconv"
)

func Fix(r io.Reader, remoteAddr string, requestId string) ([]byte, error) {
	nilVal := []byte(`- `)

	var messageWriter bytes.Buffer
	var messageLenWriter bytes.Buffer

	lp := lpx.NewReader(bufio.NewReader(r))
	for lp.Next() {
		header := lp.Header()

		// LEN SP PRI VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID SP STRUCTURED-DATA MSG
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
		messageWriter.WriteString(remoteAddr)
		messageWriter.WriteString("\"]")

		b := lp.Bytes()
		if len(b) >= 2 && bytes.Equal(b[0:2], nilVal) {
			messageWriter.Write(b[1:])
		} else if len(b) > 0 {
			if b[0] != '[' {
				messageWriter.WriteString(" ")
			}
			messageWriter.Write(b)
		}

		messageLenWriter.WriteString(strconv.Itoa(messageWriter.Len()))
		messageLenWriter.WriteString(" ")
		messageWriter.WriteTo(&messageLenWriter)
	}

	if lp.Err() != nil {
		Logf("measure.log-iss.fixer.fix.error.lpx=1 request_id=%q message=%q", requestId, lp.Err())
		return nil, lp.Err()
	}

	if fullMessage, err := ioutil.ReadAll(&messageLenWriter); err != nil {
		Logf("measure.log-iss.fixer.fix.error.readall=1 request_id=%q message=%q", requestId, err)
		return nil, err
	} else {
		return fullMessage, nil
	}
}
