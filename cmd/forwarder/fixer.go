package main

import (
	"bufio"
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bmizerany/lpx"
)

const (
	// LogplexDefaultHost is the default host from logplex:
	// https://github.com/heroku/logplex/blob/master/src/logplex_http_drain.erl#L443
	logplexDefaultHost = "host"

	maxHostnameLength = 255
	maxAppnameLength  = 48
	maxProcidLength   = 128
	maxMsgidLength    = 32
)

var nilVal = []byte("- ")
var queryParams = []string{"index", "source", "sourcetype"}

// Get metadata from the http request.
// Returns an empty byte array if there isn't any.
func getMetadata(req *http.Request, cred *credential, metadataId string, queryFieldParams []string) (string, bool) {
	var metadataWriter strings.Builder
	var foundMetadata bool

	// Calculate metadata query parameters
	if metadataId != "" {
		var fieldsBuilder strings.Builder

		// Pre-grow to minimize extra slice allocations
		metadataWriter.Grow(1024)
		fieldsBuilder.Grow(256)

		metadataWriter.WriteString("[")
		metadataWriter.WriteString(metadataId)

		for _, k := range append(queryParams, queryFieldParams...) {
			v := req.FormValue(k)
			if v != "" {
				if containsString(queryFieldParams, k) {
					if fieldsBuilder.Len() > 0 {
						fieldsBuilder.WriteString(",")
					}
					fieldsBuilder.WriteString(k)
					fieldsBuilder.WriteString("=")
					fieldsBuilder.WriteString(v)
				} else {
					metadataWriter.WriteString(" ")
					metadataWriter.WriteString(k)
					metadataWriter.WriteString(`="`)
					metadataWriter.WriteString(v)
					metadataWriter.WriteString(`"`)
				}
				foundMetadata = true
			}
		}

		// Add metadata about the credential if it is deprecated
		if cred != nil && cred.Deprecated {
			if fieldsBuilder.Len() > 0 {
				fieldsBuilder.WriteString(",")
			}
			fieldsBuilder.WriteString(`credential_deprecated=true,credential_name=`)
			fieldsBuilder.WriteString(cred.Name)
			foundMetadata = true
		}

		if foundMetadata {
			if fieldsBuilder.Len() > 0 {
				metadataWriter.WriteString(` fields="`)
				metadataWriter.WriteString(fieldsBuilder.String())
				metadataWriter.WriteString(`"`)
			}

			metadataWriter.WriteString("]")
		}
	}

	return metadataWriter.String(), foundMetadata
}

// Write a header field into the messageWriter buffer. Truncates to maxLength
// Returns true if the input string was truncated, and false otherwise.
func writeField(messageWriter *bytes.Buffer, str []byte, maxLength int) bool {
	if len(str) > maxLength {
		messageWriter.Write(str[0:maxLength])
		return true
	} else {
		messageWriter.Write(str)
		return false
	}
}

type fixResult struct {
	hasMetadata    bool
	numLogs        int64
	bytes          []byte
	hostnameTruncs int64
	appnameTruncs  int64
	procidTruncs   int64
	msgidTruncs    int64
}

// Fix function to convert post data to length prefixed syslog frames
// Returns:
// * boolean indicating whether metadata was present in the query parameters.
// * integer representing the number of logplex frames parsed from the HTTP request.
// * byte array of syslog data.
// * error if something went wrong.
func fix(req *http.Request, r io.Reader, remoteAddr string, logplexDrainToken string, metadataId string, cred *credential, queryFieldParams []string) (fixResult, error) {
	var messageWriter bytes.Buffer
	var messageLenWriter bytes.Buffer

	metadataString, hasMetadata := getMetadata(req, cred, metadataId, queryFieldParams)

	lp := lpx.NewReader(bufio.NewReader(r))
	numLogs := int64(0)
	hostnameTruncs := int64(0)
	appnameTruncs := int64(0)
	procidTruncs := int64(0)
	msgidTruncs := int64(0)
	for lp.Next() {
		numLogs++
		header := lp.Header()

		// LEN SP PRI VERSION SP TIMESTAMP SP HOSTNAME SP APP-NAME SP PROCID SP MSGID SP STRUCTURED-DATA MSG
		messageWriter.Write(header.PrivalVersion)
		messageWriter.WriteString(" ")
		messageWriter.Write(header.Time)
		messageWriter.WriteString(" ")
		host := header.Hostname
		if string(header.Hostname) == logplexDefaultHost && logplexDrainToken != "" {
			host = []byte(logplexDrainToken)
		}
		if writeField(&messageWriter, host, maxHostnameLength) {
			hostnameTruncs++
		}
		messageWriter.WriteString(" ")
		if writeField(&messageWriter, header.Name, maxAppnameLength) {
			appnameTruncs++
		}
		messageWriter.WriteString(" ")
		if writeField(&messageWriter, header.Procid, maxProcidLength) {
			procidTruncs++
		}
		messageWriter.WriteString(" ")
		if writeField(&messageWriter, header.Msgid, maxMsgidLength) {
			msgidTruncs++
		}
		messageWriter.WriteString(" ")
		if remoteAddr != "" {
			messageWriter.WriteString("[origin ip=\"")
			messageWriter.WriteString(remoteAddr)
			messageWriter.WriteString("\"]")
		}

		// Write metadata
		if hasMetadata {
			messageWriter.WriteString(metadataString)
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

	return fixResult{
		hasMetadata:    hasMetadata,
		numLogs:        numLogs,
		bytes:          messageLenWriter.Bytes(),
		hostnameTruncs: hostnameTruncs,
		appnameTruncs:  appnameTruncs,
		procidTruncs:   procidTruncs,
		msgidTruncs:    msgidTruncs,
	}, lp.Err()
}
