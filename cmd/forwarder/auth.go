package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis"
	metrics "github.com/rcrowley/go-metrics"
)

func newAuth(config AuthConfig, registry metrics.Registry) (*BasicAuth, error) {
	var err error

	if config.RedisUrl != "" && config.RedisKey == "" {
		err = errors.New("RedisKey must be set if RedisUrl is set")
		return nil, err
	}

	if config.RedisUrl == "" && config.Tokens == "" {
		err = errors.New("At least one of RedisUrl or Tokens must be set.")
		return nil, err
	}

	result, err := NewBasicAuthFromString(config.Tokens, config.HmacKey, registry)

	if err != nil {
		return result, err
	}

	pChanges := metrics.GetOrRegisterCounter("log-iss.auth_refresh.changes", registry)
	pFailures := metrics.GetOrRegisterCounter("log-iss.auth_refresh.failures", registry)
	pSuccesses := metrics.GetOrRegisterCounter("log-iss.auth_refresh.successes", registry)

	if config.RedisUrl != "" {
		// Parse redis db out of url
		u, err := url.Parse(config.RedisUrl)
		if err != nil {
			return result, err
		}

		var db int
		if u.Path != "" && u.Path != "/" {
			db, err = strconv.Atoi(u.Path[1:len(u.Path)])
			if err != nil {
				return result, err
			}
		}

		ui := u.User
		password := ""
		if ui != nil {
			password, _ = ui.Password()
		}
		client := redis.NewClient(
			&redis.Options{
				Addr:     u.Host,
				Password: password,
				DB:       db,
			},
		)

		// Refresh once at the start.
		// Ignore errors here - if redis is down, we don't want the process to fail to start.
		_, _ = refreshAuth(result, client, config.HmacKey, config.RedisKey, config.Tokens)

		// Refresh forever
		ticker := time.NewTicker(config.RefreshInterval)
		go func() {
			changed := false
			for _ = range ticker.C {
				_, err := refreshAuth(result, client, config.HmacKey, config.RedisKey, config.Tokens)
				if err == nil {
					pSuccesses.Inc(1)
					if changed {
						pChanges.Inc(1)
					}
				} else {
					fmt.Printf("Unable to refresh credentials: %s", err)
					pFailures.Inc(1)
				}
			}
		}()
	}

	return result, err
}

// Refresh auth credentials.
// Return true if credentials changed, false otherwise.
func refreshAuth(ba *BasicAuth, client redis.Cmdable, hmacKey string, redisKey string, config string) (bool, error) {
	// Start out using the strings from config
	nba, err := NewBasicAuthFromString(config, hmacKey, ba.registry)
	if err != nil {
		return false, err
	}

	// Retrieve all secrets and construct a string in the format expected by BasicAuth
	val := client.HGetAll(redisKey)
	r, err := val.Result()
	if err != nil {
		return false, err
	}

	for k, v := range r {
		var arr []Credential
		err = json.Unmarshal([]byte(v), &arr)

		if err != nil {
			return false, err
		}
		nba.creds[k] = arr
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
	creds    map[string][]Credential
	hmacKey  string
	registry metrics.Registry
}

// NewBasicAuthFromString creates and populates a BasicAuth from the provided
// credentials, encoded as a string, in the following format:
// user:password|user:password|...
func NewBasicAuthFromString(creds string, hmacKey string, registry metrics.Registry) (*BasicAuth, error) {
	ba := NewBasicAuth(registry, hmacKey)

	// If the string is empty, that's allowed.
	if creds == "" {
		return ba, nil
	}

	for _, u := range strings.Split(creds, "|") {
		uparts := strings.SplitN(u, ":", 2)
		if len(uparts) != 2 || len(uparts[0]) == 0 || len(uparts[1]) == 0 {
			return ba, fmt.Errorf("Unable to create credentials from '%s'", u)
		}

		value := uparts[1]
		ba.AddPrincipal(uparts[0], hmacEncode(hmacKey, value), "env")
	}
	return ba, nil
}

func hmacEncode(key string, value string) string {
	secret := []byte(key)
	message := []byte(value)
	hash := hmac.New(sha512.New, secret)
	hash.Write(message)
	return hex.EncodeToString(hash.Sum(nil))
}

func NewBasicAuth(registry metrics.Registry, hmacKey string) *BasicAuth {
	return &BasicAuth{
		creds:    make(map[string][]Credential),
		hmacKey:  hmacKey,
		registry: registry,
	}
}

// AddPrincipal add's a user/password combo to the list of valid combinations
func (ba *BasicAuth) AddPrincipal(user string, hmac string, stage string) {
	ba.Lock()
	u, existed := ba.creds[user]
	if !existed {
		u = make([]Credential, 0, 1)
	}
	ba.creds[user] = append(u, Credential{Stage: stage, Hmac: hmac})
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

	if credentials, exists := ba.creds[user]; exists {
		for _, c := range credentials {
			if c.Hmac == hmacEncode(ba.hmacKey, pass) {
				countName := fmt.Sprintf("log-iss.auth.successes.%s.%s", user, c.Stage)
				counter := metrics.GetOrRegisterCounter(countName, ba.registry)
				counter.Inc(1)
				return true
			}
		}
		countName := fmt.Sprintf("log-iss.auth.failures.%s", user)
		counter := metrics.GetOrRegisterCounter(countName, ba.registry)
		counter.Inc(1)
	}

	return false
}
