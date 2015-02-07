package main

import (
	"fmt"
	"html/template"
	"net/http"
	"os"

	logiss "github.com/heroku/log-iss"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/aws"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/awslabs/aws-sdk-go/gen/dynamodb"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/freeformz/googlegoauth"
	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/kr/secureheader"
)

const (
	dynamoDBTableName = "log-iss-users"
	region            = "us-east-1"
)

var (
	index = template.Must(template.ParseFiles("cmd/admin/ui/_base.tmpl", "cmd/admin/ui/index.tmpl"))
	there = template.Must(template.ParseFiles("cmd/admin/ui/_base.tmpl", "cmd/admin/ui/there.tmpl"))
	add   = template.Must(template.ParseFiles("cmd/admin/ui/_base.tmpl", "cmd/admin/ui/add.tmpl"))

	creds = aws.Creds(os.Getenv("AWS_KEY"), os.Getenv("AWS_SECRET"), "")
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if err := index.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func thereHandler(w http.ResponseWriter, r *http.Request) {
	if err := there.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func addHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	type user struct {
		User, Password, URL string
	}

	u := user{
		User:     r.FormValue("name"),
		Password: r.FormValue("password"),
		URL:      "google.com/logs",
	}

	ddb := dynamodb.New(creds, region, nil)

	ddbreq := logiss.NewUserItem(dynamoDBTableName, u.User, u.Password, r.FormValue("note"))
	ddbresp, err := ddb.PutItem(&ddbreq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	fmt.Printf("%+q\n", ddbresp)

	if err := add.Execute(w, u); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {

	mux := http.NewServeMux()

	mux.HandleFunc("/", indexHandler)
	mux.HandleFunc("/there/", thereHandler)
	mux.HandleFunc("/add/", addHandler)

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
