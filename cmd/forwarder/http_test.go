package main

import "testing"

func TestContentTypeValdidation(t *testing.T) {
	srv := &httpServer{}

	var cases = []struct {
		in  string
		out bool
	}{
		{"", false},
		{ctLogplexV1, true},
		{ctMsgpack, true},
	}

	for _, tc := range cases {
		if srv.validContentType(tc.in) != tc.out {
			t.Errorf("got %t; want %t", srv.validContentType(tc.in), tc.out)
		}
	}

}
