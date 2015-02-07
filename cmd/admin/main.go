package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/freeformz/googlegoauth"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/kr/secureheader"
)

var index = template.Must(template.ParseFiles("cmd/admin/ui/_base.tmpl", "cmd/admin/ui/index.tmpl"))
var there = template.Must(template.ParseFiles("cmd/admin/ui/_base.tmpl", "cmd/admin/ui/there.tmpl"))

func listData(w http.ResponseWriter, r *http.Request) {
	if err := index.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func thereHandler(w http.ResponseWriter, r *http.Request) {
	if err := there.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", listData)
	mux.HandleFunc("/there/", thereHandler)

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
