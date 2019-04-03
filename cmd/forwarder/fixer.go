package main

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/bmizerany/lpx"
)

const (
	// LogplexDefaultHost is the default host from logplex:
	// https://github.com/heroku/logplex/blob/master/src/logplex_http_drain.erl#L443
	logplexDefaultHost = "host"
)

var nilVal = []byte(`- `)
var queryParams = []string{"index", "source", "sourcetype"}

var pool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

func getBuffer() *bytes.Buffer {
	return pool.Get().(*bytes.Buffer)
}

func putBuffer(b *bytes.Buffer) {
	b.Reset() // ensure that the buffer is ready for re-use
	pool.Put(b)
}

// Get metadata from the http request.
// Returns an empty byte array if there isn't any.
func getMetadata(req *http.Request, cred *credential, metadataId string) []byte {
	if metadataId == "" { // short circuit when metadataId is empty
		return nil
	}

	metadataWriter := getBuffer()
	defer putBuffer(metadataWriter)
	writeSDElementStart := func() {
		metadataWriter.WriteString("[")
		metadataWriter.WriteString(metadataId)
	}

	// Calculate metadata query parameters
	for _, k := range queryParams {
		if v := req.FormValue(k); v != "" {
			if metadataWriter.Len() == 0 {
				writeSDElementStart()
			}
			metadataWriter.WriteString(" ")
			metadataWriter.WriteString(k)
			metadataWriter.WriteString("=\"")
			metadataWriter.WriteString(v)
			metadataWriter.WriteString("\"")
		}
	}

	// Add metadata about the credential if it is deprecated
	if cred != nil && cred.Deprecated == true {
		if metadataWriter.Len() == 0 {
			writeSDElementStart()
		}
		metadataWriter.WriteString(` fields="credential_deprecated=true,credential_name=`)
		metadataWriter.WriteString(cred.Name)
		metadataWriter.WriteString(`"`)
	}

	if metadataWriter.Len() > 0 {
		metadataWriter.WriteString("]")
	}
	return metadataWriter.Bytes()
}

// Fix function to convert post data to length prefixed syslog frames
// Returns:
// * boolean indicating whether metadata was present in the query parameters.
// * integer representing the number of logplex frames parsed from the HTTP request.
// * byte array of syslog data.
// * error if something went wrong.
func fix(req *http.Request, r io.Reader, remoteAddr string, logplexDrainToken string, metadataId string, cred *credential) (bool, int64, []byte, error) {
	messageWriter := getBuffer()
	messageLenWriter := getBuffer()
	defer func() {
		putBuffer(messageWriter)
		putBuffer(messageLenWriter)
	}()

	metadataBytes := getMetadata(req, cred, metadataId)

	lp := lpx.NewReader(bufio.NewReader(r))
	var numLogs int64
	for ; lp.Next(); numLogs++ {
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

		// Write metadata
		messageWriter.Write(metadataBytes)

		if b := lp.Bytes(); len(b) >= 2 && bytes.Equal(b[0:2], nilVal) {
			messageWriter.Write(b[1:])
		} else if len(b) > 0 {
			if b[0] != '[' {
				messageWriter.WriteString(" ")
			}
			messageWriter.Write(b)
		}

		messageLenWriter.WriteString(strconv.Itoa(messageWriter.Len()))
		messageLenWriter.WriteString(" ")
		messageWriter.WriteTo(messageLenWriter)
	}

	return len(metadataBytes) > 0, numLogs, messageLenWriter.Bytes(), lp.Err()
}
