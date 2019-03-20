package main

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

type InputOutput struct {
	Input  []byte
	Output []byte
}

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
		hasMetadata, _, fixed, _ := fix(simpleHttpRequest(), bytes.NewReader(in), "1.2.3.4", "", "", nil)
		assert.Equal(string(fixed), string(output[x]))
		assert.False(hasMetadata)
	}
}

func TestFixWithQueryParameters(t *testing.T) {
	assert := assert.New(t)
	var output = []byte("135 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\"] hi\n138 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\"] hello\n")

	in := input[0]
	hasMetadata, numLogs, fixed, _ := fix(httpRequestWithParams(), bytes.NewReader(in), "1.2.3.4", "", "metadata@123", nil)

	assert.Equal(string(fixed), string(output), "They should be equal")
	assert.True(hasMetadata)
	assert.Equal(int64(2), numLogs)
}

func TestFixWithPreviousCredential(t *testing.T) {
	assert := assert.New(t)
	var output = []byte("177 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\" fields=\"{'credential_stage': 'previous'}\"] hi\n180 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][metadata@123 index=\"i\" source=\"s\" sourcetype=\"st\" fields=\"{'credential_stage': 'previous'}\"] hello\n")

	in := input[0]
	cred := credential{Stage: "previous"}
	hasMetadata, numLogs, fixed, _ := fix(httpRequestWithParams(), bytes.NewReader(in), "1.2.3.4", "", "metadata@123", &cred)

	assert.Equal(string(fixed), string(output), "They should be equal")
	assert.True(hasMetadata)
	assert.Equal(int64(2), numLogs)
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
		hasMetadata, _, fixed, _ := fix(simpleHttpRequest(), bytes.NewReader(in), "1.2.3.4", testToken, "", nil)
		assert.Equal(string(fixed), string(output[x]))
		assert.False(hasMetadata)
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
