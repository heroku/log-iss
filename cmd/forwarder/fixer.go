package main

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strconv"

	"github.com/bmizerany/lpx"
)

const (
	// LogplexDefaultHost is the default host from logplex:
	// https://github.com/heroku/logplex/blob/master/src/logplex_http_drain.erl#L443
	logplexDefaultHost = "host"
)

var nilVal = []byte(`- `)
var queryParams = []string{"index", "source", "sourcetype"}

// Get metadata from the http request.
// Returns an empty byte array if there isn't any.
func getMetadata(req *http.Request, cred *credential, metadataId string) ([]byte, bool) {
	var metadataWriter bytes.Buffer
	var foundMetadata bool
	// Calculate metadata query parameters
	if metadataId != "" {
		for _, k := range queryParams {
			v := req.FormValue(k)
			if v != "" {
				if !foundMetadata {
					metadataWriter.WriteString("[")
					metadataWriter.WriteString(metadataId)
					foundMetadata = true
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
			if !foundMetadata {
				metadataWriter.WriteString("[")
				metadataWriter.WriteString(metadataId)
				foundMetadata = true
			}
			metadataWriter.WriteString(` fields="credential_deprecated=true,credential_name=`)
			metadataWriter.WriteString(cred.Name)
			metadataWriter.WriteString(`"`)
		}

		if foundMetadata {
			metadataWriter.WriteString("]")
		}
	}
	return metadataWriter.Bytes(), foundMetadata
}

// Fix function to convert post data to length prefixed syslog frames
// Returns:
// * boolean indicating whether metadata was present in the query parameters.
// * integer representing the number of logplex frames parsed from the HTTP request.
// * byte array of syslog data.
// * error if something went wrong.
func fix(req *http.Request, r io.Reader, remoteAddr string, logplexDrainToken string, metadataId string, cred *credential) (bool, int64, []byte, error) {
	var messageWriter bytes.Buffer
	var messageLenWriter bytes.Buffer

	metadataBytes, hasMetadata := getMetadata(req, cred, metadataId)

	lp := lpx.NewReader(bufio.NewReader(r))
	numLogs := int64(0)
	for lp.Next() {
		numLogs++
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
		if hasMetadata {
			messageWriter.Write(metadataBytes)
		}

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

	return hasMetadata, numLogs, messageLenWriter.Bytes(), lp.Err()
}
