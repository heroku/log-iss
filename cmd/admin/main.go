package main

import (
	"html/template"
	"net/http"
	"os"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/freeformz/googlegoauth"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/kr/secureheader"
)

var templates = template.Must(template.ParseFiles("view.html"))

func listData(w http.ResponseWriter, r *http.Request) {
	err := templates.ExecuteTemplate(w, "view.html", nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
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
