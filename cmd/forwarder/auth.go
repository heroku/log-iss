package main

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"

	smi "github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	metrics "github.com/rcrowley/go-metrics"
)

func newAuth(config AuthConfig, registry metrics.Registry) (BasicAuth, error) {
	result, err := NewBasicAuthFromString(config.Tokens)
	if err != nil {
		return *result, err
	}

	pChanges := metrics.GetOrRegisterCounter("log-iss.auth_refresh.changes", registry)
	pFailures := metrics.GetOrRegisterCounter("log-iss.auth_refresh.failures", registry)
	pSuccesses := metrics.GetOrRegisterCounter("log-iss.auth_refresh.successes", registry)

	ticker := time.NewTicker(config.RefreshInterval)
	go func() {
		sess := session.Must(session.NewSession())
		client := secretsmanager.New(sess)
		for _ = range ticker.C {
			changed, err := refreshAuth(result, client, config.SecretPrefix, config.Tokens)
			if err == nil {
				pSuccesses.Inc(1)
				if changed {
					pChanges.Inc(1)
				}
			} else {
				pFailures.Inc(1)
			}
		}
	}()

	return *result, err
}

// Refresh auth credentials.
// Return true if credentials changed, false otherwise.
func refreshAuth(ba *BasicAuth, client smi.SecretsManagerAPI, prefix string, config string) (bool, error) {
	// Start out using the strings from config
	prefixLength := len(prefix)
	nba, err := NewBasicAuthFromString(config)
	if err != nil {
		return false, err
	}

	// Retrieve all secrets and construct a string in the format expected by BasicAuth
	err = client.ListSecretsPages(nil,
		func(page *secretsmanager.ListSecretsOutput, lastPage bool) bool {
			for _, sle := range page.SecretList {
				if strings.HasPrefix(*(sle.Name), prefix) {
					user := (*sle.Name)[prefixLength+1 : len(*sle.Name)]
					gsvi := secretsmanager.GetSecretValueInput{
						SecretId: sle.ARN,
					}
					gsvo, err := client.GetSecretValue(&gsvi)
					if err != nil {
						return true
					}
					nba.creds[user] = []string{*gsvo.SecretString}
				}
			}
			return page.NextToken == nil
		})
	if err != nil {
		return false, err
	}

	// Swap the secrets if there are changes
	if !reflect.DeepEqual(ba.creds, nba.creds) {
		ba.Lock()
		defer ba.Unlock()
		ba.creds = nba.creds
		return true, nil
	}
	return false, nil
}

// BasicAuth handles normal user/password Basic Auth requests, multiple
// password for the same user and is safe for concurrent use.
type BasicAuth struct {
	sync.RWMutex
	creds map[string][]string
}

// NewBasicAuthFromString creates and populates a BasicAuth from the provided
// credentials, encoded as a string, in the following format:
// user:password|user:password|...
func NewBasicAuthFromString(creds string) (*BasicAuth, error) {
	ba := NewBasicAuth()
	for _, u := range strings.Split(creds, "|") {
		uparts := strings.SplitN(u, ":", 2)
		if len(uparts) != 2 || len(uparts[0]) == 0 || len(uparts[1]) == 0 {
			return ba, fmt.Errorf("Unable to create credentials from '%s'", u)
		}

		ba.AddPrincipal(uparts[0], uparts[1])
	}
	return ba, nil
}

func NewBasicAuth() *BasicAuth {
	return &BasicAuth{
		creds: make(map[string][]string),
	}
}

// AddPrincipal add's a user/password combo to the list of valid combinations
func (ba *BasicAuth) AddPrincipal(user, pass string) {
	ba.Lock()
	u, existed := ba.creds[user]
	if !existed {
		u = make([]string, 0, 1)
	}
	ba.creds[user] = append(u, pass)
	ba.Unlock()
}

// Authenticate is true if the Request has a valid BasicAuth signature and
// that signature encodes a known username/password combo.
func (ba *BasicAuth) Authenticate(r *http.Request) bool {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return false
	}

	ba.RLock()
	defer ba.RUnlock()

	if passwords, exists := ba.creds[user]; exists {
		for _, password := range passwords {
			if password == pass {
				return true
			}
		}
	}

	return false
}
