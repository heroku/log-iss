package main

import (
	"net/http"
	"os"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/freeformz/googlegoauth"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/kr/secureheader"
)

func main() {

	behindGoogleAuth := &googlegoauth.Handler{
		RequireDomain: os.Getenv("REQUIRE_DOMAIN"),
		Key:           os.Getenv("KEY"),
		ClientID:      os.Getenv("CLIENT_ID"),
		ClientSecret:  os.Getenv("CLIENT_SECRET"),
		Handler:       http.FileServer(http.Dir(".")),
	}

	http.Handle("/", behindGoogleAuth)

	http.ListenAndServe(":"+os.Getenv("PORT"), secureheader.DefaultConfig)
}
