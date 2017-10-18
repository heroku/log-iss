package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heroku/authenticater"
)

func TestHandleLogs(t *testing.T) {
	cfg := &IssConfig{}
	auth := authenticater.AnyOrNoAuth{}
	fwd := &nullForwarder{}
	srv := newHTTPServer(*cfg, auth, fwd)

	handler := http.HandlerFunc(srv.handleLogs)
	s := httptest.NewServer(handler)
	defer s.Close()

	logReq := func(contentType string, body io.Reader) *http.Request {
		r, err := http.NewRequest("POST", s.URL, body)
		r.Header.Set("Content-Type", contentType)
		if err != nil {
			t.Fatal(err)
		}

		return r
	}

	t.Run("Content Types", func(t *testing.T) {
		cases := []struct {
			name   string
			r      *http.Request
			status int
		}{
			{"Empty String", logReq("", nil), 400},
			{"Logplex", logReq(ctLogplexV1, nil), 200},
			{"msgpack", logReq(ctMsgpack, nil), 200},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				resp, err := http.DefaultClient.Do(tc.r)
				defer resp.Body.Close()
				if err != nil {
					t.Fatal(err)
				}

				if resp.StatusCode != tc.status {
					t.Errorf("got %d; want %d", resp.StatusCode, tc.status)
				}
			})
		}
	})
}
