package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func StartHttp(port string, tokens Tokens, outlet chan []byte) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// check outlet depth?
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Only POST is accepted", 400)
			return
		}
		if r.Header.Get("Content-Type") != "application/logplex-1" {
			http.Error(w, "Only Content-Type application/logplex-1 is accepted", 400)
			return
		}

		err := checkAuth(r, tokens)
		if err != nil {
			http.Error(w, err.Error(), 401)
			return
		}

		b, err := ioutil.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			http.Error(w, "Invalid Request", 400)
			return
		}

		outlet <- b
	})

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalln("Unable to start HTTP server:", err)
	}
}

func checkAuth(r *http.Request, tokens Tokens) error {
	header := r.Header.Get("Authorization")
	if header == "" {
		return errors.New("Authorization required")
	}
	headerParts := strings.SplitN(header, " ", 2)
	if len(headerParts) != 2 {
		return errors.New("Authorization header is malformed")
	}

	method := headerParts[0]
	if method != "Basic" {
		return errors.New("Only Basic Authorization is accepted")
	}

	encodedUserPass := headerParts[1]
	decodedUserPass, err := base64.StdEncoding.DecodeString(encodedUserPass)
	if err != nil {
		return errors.New("Authorization header is malformed")
	}

	userPassParts := bytes.SplitN(decodedUserPass, []byte{':'}, 2)
	if len(userPassParts) != 2 {
		return errors.New("Authorization header is malformed")
	}

	user := userPassParts[0]
	pass := userPassParts[1]
	token, ok := tokens[string(user)]
	if !ok {
		return errors.New("Unknown user")
	}
	if token != string(pass) {
		return errors.New("Incorrect token")
	}

	return nil
}
