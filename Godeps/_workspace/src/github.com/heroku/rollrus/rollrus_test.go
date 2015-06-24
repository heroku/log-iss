package rollrus

import (
	"fmt"
	"testing"
	"time"

	log "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

func TestIntConversion(t *testing.T) {
	i := make(log.Fields)
	i["test"] = 5

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "5" {
		t.Fatal("Expected value to equal 5, but instead it is: ", v)
	}
}

func TestErrConversion(t *testing.T) {
	i := make(log.Fields)
	i["test"] = fmt.Errorf("This is an error")

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "This is an error" {
		t.Fatal("Expected value to be a string of the error but instead it is: ", v)
	}
}

func TestStringConversion(t *testing.T) {
	i := make(log.Fields)
	i["test"] = "This is a string"

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != "This is a string" {
		t.Fatal("Expected value to equal a certain string, but instead it is: ", v)
	}
}

func TestTimeConversion(t *testing.T) {
	now := time.Now()
	i := make(log.Fields)
	i["test"] = now

	r := convertFields(i)

	v, ok := r["test"]
	if !ok {
		t.Fatal("Expected test key, but did not find it")
	}

	if v != now.Format(time.RFC3339) {
		t.Fatal("Expected value to equal, but instead it is: ", v)
	}
}
