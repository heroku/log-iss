package main

import (
	"net/http"
	"os"

	"github.com/freeformz/googlegoauth"
	"github.com/kr/secureheader"
)

func listData(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", listData)

	behindGoogleAuth := &googlegoauth.Handler{
		RequireDomain: os.Getenv("REQUIRE_DOMAIN"),
		Key:           os.Getenv("KEY"),
		ClientID:      os.Getenv("CLIENT_ID"),
		ClientSecret:  os.Getenv("CLIENT_SECRET"),
		Handler:       mux,
	}

	http.Handle("/", behindGoogleAuth)

	http.ListenAndServe(":"+os.Getenv("PORT"), secureheader.DefaultConfig)
}
