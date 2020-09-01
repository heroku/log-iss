package main

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	input = [][]byte{
		[]byte("64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi\n67 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hello\n"),
		[]byte("106 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
		[]byte("65 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - hello\n"),
		[]byte("58 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - "),
		[]byte("97 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [60607e20-f12d-483e-aa89-ffaf954e7527]"),
	}
)

func TestFix(t *testing.T) {
	assert := assert.New(t)
	var output = [][]byte{
		[]byte("84 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hi\n87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("127 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
		[]byte("87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("80 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"]"),
		[]byte("118 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][60607e20-f12d-483e-aa89-ffaf954e7527]"),
	}

	for x, in := range input {
		r, _ := fix(simpleHttpRequest(), bytes.NewReader(in), "1.2.3.4", "", "", nil)
		assert.Equal(string(output[x]), string(r.bytes))
		assert.False(r.hasMetadata)
	}
}

func TestTruncationOfFields(t *testing.T) {
	assert := assert.New(t)
	type input struct {
		name        string
		bytes       []byte
		expected    []byte
		hasMetadata bool
		err         error
	}
	tests := []input{
		{
			name:        "truncate HOSTNAME",
			bytes:       []byte(fmt.Sprintf("311 <13>1 2013-06-07T13:17:49.468822+00:00 %s heroku web.7 - ", strings.Repeat("a", 256))),
			expected:    []byte(fmt.Sprintf("310 <13>1 2013-06-07T13:17:49.468822+00:00 %s heroku web.7 - ", strings.Repeat("a", 255))),
			hasMetadata: false,
		},
		{
			name:        "truncate APP-NAME",
			bytes:       []byte(fmt.Sprintf("102 <13>1 2013-06-07T13:17:49.468822+00:00 host %s web.7 - ", strings.Repeat("a", 49))),
			expected:    []byte(fmt.Sprintf("101 <13>1 2013-06-07T13:17:49.468822+00:00 host %s web.7 - ", strings.Repeat("a", 48))),
			hasMetadata: false,
		},
		{
			name:        "truncate PROCID",
			bytes:       []byte(fmt.Sprintf("183 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku %s - ", strings.Repeat("a", 129))),
			expected:    []byte(fmt.Sprintf("182 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku %s - ", strings.Repeat("a", 128))),
			hasMetadata: false,
		},
		{
			name:        "truncate MSGID",
			bytes:       []byte(fmt.Sprintf("91 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 %s ", strings.Repeat("a", 33))),
			expected:    []byte(fmt.Sprintf("90 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 %s ", strings.Repeat("a", 32))),
			hasMetadata: false,
		},
	}

	for _, i := range tests {
		t.Run(i.name, func(t *testing.T) {
			r, err := fix(simpleHttpRequest(), bytes.NewReader(i.bytes), "", "", "", nil)

			assert.Equal(i.err, err)
			assert.Equal(string(i.expected), string(r.bytes))
			assert.Equal(i.hasMetadata, r.hasMetadata)
		})
	}
}

func TestFixWithQueryParameters(t *testing.T) {
	assert := assert.New(t)
	var output = []byte("135 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\"] hi\n138 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\"] hello\n")

	in := input[0]
	r, _ := fix(httpRequestWithParams(), bytes.NewReader(in), "1.2.3.4", "", "metadata@123", nil)

	assert.Equal(string(output), string(r.bytes))
	assert.True(r.hasMetadata)
	assert.Equal(int64(2), r.numLogs)
}

func TestFixWithDeprecatedCredential(t *testing.T) {
	assert := assert.New(t)
	var output = []byte("192 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\" fields=\"credential_deprecated=true,credential_name=cred\"] hi\n195 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\" fields=\"credential_deprecated=true,credential_name=cred\"] hello\n")

	in := input[0]
	cred := credential{Stage: "previous", Name: "cred", Deprecated: true}
	r, _ := fix(httpRequestWithParams(), bytes.NewReader(in), "1.2.3.4", "", "metadata@123", &cred)

	assert.Equal(string(output), string(r.bytes))
	assert.True(r.hasMetadata)
	assert.Equal(r.numLogs, int64(2))
}

func TestFixWithLogplexDrainToken(t *testing.T) {
	assert := assert.New(t)
	testToken := "d.34bc219c-983b-463e-a17d-3d34ee7db812"

	output := [][]byte{
		[]byte("118 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hi\n121 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("161 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"][meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
		[]byte("121 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("114 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"]"),
		[]byte("152 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"][60607e20-f12d-483e-aa89-ffaf954e7527]"),
	}
	for x, in := range input {
		r, _ := fix(simpleHttpRequest(), bytes.NewReader(in), "1.2.3.4", testToken, "", nil)
		assert.Equal(string(output[x]), string(r.bytes))
		assert.False(r.hasMetadata)
	}
}

func BenchmarkFixNoSD(b *testing.B) {
	input := []byte("64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi\n67 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hello\n")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fix(simpleHttpRequest(), bytes.NewReader(input), "1.2.3.4", "", "", nil)
	}
}

func BenchmarkFixSD(b *testing.B) {
	input := []byte("106 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		fix(simpleHttpRequest(), bytes.NewReader(input), "1.2.3.4", "", "", nil)
	}
}

func httpRequestWithParams() *http.Request {
	req, _ := http.NewRequest("POST", "/logs?index=i&source=s&sourcetype=st", nil)
	return req
}

func simpleHttpRequest() *http.Request {
	req, _ := http.NewRequest("POST", "/logs", nil)
	return req
}
