package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/amerine/msgpack-dumper/decoder"
	"github.com/bmizerany/lpx"
)

const (
	// LogplexDefaultHost is the default host from logplex:
	// https://github.com/heroku/logplex/blob/master/src/logplex_http_drain.erl#L443
	logplexDefaultHost = "host"
	rfc3339Micro       = "2006-01-02T15:04:05.999999Z07:00"
)

var nilVal = []byte(`- `)

// logplexToSyslog converts post data logplex messages to length prefixed syslog frames
func logplexToSyslog(r io.Reader, remoteAddr string, logplexDrainToken string) ([]byte, error) {
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
		if string(header.Hostname) == logplexDefaultHost && logplexDrainToken != "" {
			messageWriter.WriteString(logplexDrainToken)
		} else {
			messageWriter.Write(header.Hostname)
		}
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

	return messageLenWriter.Bytes(), lp.Err()
}

// msgpackToSyslog converts post data msgpack messages to length prefixed syslog frames
func msgpackToSyslog(r io.Reader, remoteAddr string, logplexDrainToken string) ([]byte, error) {
	var messageWriter bytes.Buffer
	var messageLenWriter bytes.Buffer

	// LEN SP <PRI>VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID SP STRUCTURED-DATA MSG
	dec := decoder.NewDecoder(r)
	for {
		ret, ts, rec := decoder.GetRecord(dec)
		if ret != 0 {
			break // No more records
		}

		timestamp := ts.(decoder.FLBTime)
		messageWriter.WriteString("<")
		messageWriter.WriteString(fetchValues(rec, "PRIORITY"))
		messageWriter.WriteString(">1")
		messageWriter.WriteString(" ")
		messageWriter.WriteString(timestamp.Format(rfc3339Micro))
		messageWriter.WriteString(" ")
		if logplexDrainToken != "" {
			messageWriter.WriteString(logplexDrainToken)
		} else {
			messageWriter.WriteString(fetchValues(rec, "_HOSTNAME", "HOSTNAME"))
		}
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(rec, "SYSLOG_IDENTIFIER", "_COMM"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(rec, "_PID"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(rec, "MESSAGE_ID"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString("[origin ip=\"")
		messageWriter.WriteString(remoteAddr)
		messageWriter.WriteString("\"]")
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(rec, "MESSAGE"))
		messageWriter.WriteString("\n")
		messageLenWriter.WriteString(strconv.Itoa(messageWriter.Len()))
		messageLenWriter.WriteString(" ")
		messageWriter.WriteTo(&messageLenWriter)
	}

	return messageLenWriter.Bytes(), nil
}

func fetchValues(data map[interface{}]interface{}, fields ...string) string {
	for k, v := range data {
		for _, f := range fields {
			if f == fmt.Sprintf("%s", k) {
				return fmt.Sprintf("%s", v)
			}
		}
	}

	return "-"
}
