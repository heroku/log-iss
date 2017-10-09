package main

import (
	"bytes"
	"testing"

	"github.com/amerine/msgpack-dumper/decoder"
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

func TestLogplexToSyslog(t *testing.T) {
	var output = [][]byte{
		[]byte("84 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hi\n87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("127 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
		[]byte("87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("80 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"]"),
		[]byte("118 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][60607e20-f12d-483e-aa89-ffaf954e7527]"),
	}
	for x, in := range input {
		fixed, _ := logplexToSyslog(bytes.NewReader(in), "1.2.3.4", "")

		if !bytes.Equal(fixed, output[x]) {
			t.Errorf("input=%q\noutput=%q\ngot=%q\n", in, output[x], fixed)
		}
	}
}

func TestLogplexToSyslogWithLogplexDrainToken(t *testing.T) {
	testToken := "d.34bc219c-983b-463e-a17d-3d34ee7db812"

	output := [][]byte{
		[]byte("118 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hi\n121 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("161 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"][meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
		[]byte("121 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
		[]byte("114 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"]"),
		[]byte("152 <13>1 2013-06-07T13:17:49.468822+00:00 d.34bc219c-983b-463e-a17d-3d34ee7db812 heroku web.7 - [origin ip=\"1.2.3.4\"][60607e20-f12d-483e-aa89-ffaf954e7527]"),
	}

	for x, in := range input {
		fixed, _ := logplexToSyslog(bytes.NewReader(in), "1.2.3.4", testToken)

		if !bytes.Equal(fixed, output[x]) {
			t.Errorf("input=%q\noutput=%q\ngot=%q\n", in, output[x], fixed)
		}
	}
}

func BenchmarkLogplexToSyslogNoSD(b *testing.B) {
	input := []byte("64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi\n67 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hello\n")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logplexToSyslog(bytes.NewReader(input), "1.2.3.4", "")
	}
}

func BenchmarkLogplexToSyslogSD(b *testing.B) {
	input := []byte("106 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n")
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logplexToSyslog(bytes.NewReader(input), "1.2.3.4", "")
	}
}

func TestMsgpackToSyslog(t *testing.T) {
	want := []byte("195 <6>1 2017-10-05T17:55:40.537067+00:00 ip-10-0-5-33 kubelet 8131 - [origin ip=\"1.2.3.4\"] I1005 17:55:40.799530    8131 server.go:794] GET /pods: (493.06µs) 200 [[Go-http-client/1.1] [::1]:48602]\n")
	rdr := bytes.NewReader(decoder.ExampleMessage)

	got, err := msgpackToSyslog(rdr, "1.2.3.4", "")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("got [%#v]; want [%#v]", string(got), string(want))
	}
}

func BenchmarkMsgpackToSyslogSD(b *testing.B) {
	b.SetBytes(int64(len(decoder.ExampleMessage)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		msgpackToSyslog(bytes.NewReader(decoder.ExampleMessage), "1.2.3.4", "")
	}
}
