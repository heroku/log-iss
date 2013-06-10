package main

import (
	"bytes"
	"testing"
)

type InputOutput struct {
	Input  []byte
	Output [][]byte
}

func TestFix(t *testing.T) {
	var inputOutput []InputOutput = []InputOutput{
		InputOutput{
			[]byte("64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi\n67 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hello\n"),
			[][]byte{
				[]byte("84 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hi\n"),
				[]byte("87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
			},
		},
		InputOutput{
			[]byte("106 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
			[][]byte{
				[]byte("127 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"][meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n"),
			},
		},
		InputOutput{
			[]byte("65 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - hello\n"),
			[][]byte{
				[]byte("87 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [origin ip=\"1.2.3.4\"] hello\n"),
			},
		},
	}

	for _, io := range inputOutput {
		payload := Payload{"1.2.3.4", io.Input}
		fixedMessages := Fix(payload)

		if len(fixedMessages) != len(io.Output) {
			t.Fatalf("input=%q len_output=%d got=%d\n", io.Input, len(io.Output), len(fixedMessages))
		}

		for i, m := range fixedMessages {
			if !bytes.Equal(m, io.Output[i]) {
				t.Errorf("i=%d input=%q output=%q got=%q\n", i, io.Input, io.Output[i], m)
			}
		}
	}
}

func BenchmarkFixNoSD(b *testing.B) {
	input := []byte("64 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hi\n67 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - - hello\n")
	payload := Payload{"1.2.3.4", input}
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Fix(payload)
	}
}

func BenchmarkFixSD(b *testing.B) {
	input := []byte("106 <13>1 2013-06-07T13:17:49.468822+00:00 host heroku web.7 - [meta sequenceId=\"hello\"][foo bar=\"baz\"] hello\n")
	payload := Payload{"1.2.3.4", input}
	b.SetBytes(int64(len(input)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Fix(payload)
	}
}
