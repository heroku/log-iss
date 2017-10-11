package main

import (
	"bufio"
	"bytes"
	"io"
	"strconv"

	"github.com/amerine/msgpack-dumper/decoder"
	"github.com/bmizerany/lpx"
)

const (
	// LogplexDefaultHost is the default host from logplex:
	// https://github.com/heroku/logplex/blob/master/src/logplex_http_drain.erl#L443
	logplexDefaultHost = "host"

	// logplexBatchTimeFormat is the format of timestamps as expected by Logplex. This is the structure log-shuttle sends to log-iss.
	logplexBatchTimeFormat = "2006-01-02T15:04:05.000000+00:00"
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
		rec, err := dec.GetRecord()
		if err == io.EOF {
			break // No more records
		}

		d, err := decoder.ExtractData(rec)
		ts, err := decoder.ExtractTime(rec)
		if err != nil {
			return nil, err
		}

		messageWriter.WriteString("<")
		messageWriter.WriteString(fetchValues(d, "PRIORITY"))
		messageWriter.WriteString(">1")
		messageWriter.WriteString(" ")
		messageWriter.WriteString(ts.UTC().Format(logplexBatchTimeFormat))
		messageWriter.WriteString(" ")
		if logplexDrainToken != "" {
			messageWriter.WriteString(logplexDrainToken)
		} else {
			messageWriter.WriteString(fetchValues(d, "_HOSTNAME", "HOSTNAME"))
		}
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(d, "SYSLOG_IDENTIFIER", "_COMM"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(d, "_PID"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(d, "MESSAGE_ID"))
		messageWriter.WriteString(" ")
		messageWriter.WriteString("[origin ip=\"")
		messageWriter.WriteString(remoteAddr)
		messageWriter.WriteString("\"]")
		messageWriter.WriteString(" ")
		messageWriter.WriteString(fetchValues(d, "MESSAGE"))
		messageWriter.WriteString("\n")
		messageLenWriter.WriteString(strconv.Itoa(messageWriter.Len()))
		messageLenWriter.WriteString(" ")
		messageWriter.WriteTo(&messageLenWriter)
	}

	return messageLenWriter.Bytes(), nil
}

func fetchValues(data map[string]interface{}, fields ...string) string {
	for _, f := range fields {
		if _, ok := data[f]; ok {
			return data[f].(string)
		}
	}

	return "-"
}
