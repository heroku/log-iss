package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/elliotchance/redismock"
	"github.com/go-redis/redis"
	metrics "github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
)

func noSecretsRedis() redis.Cmdable {
	r := redismock.NewMock()
	r.On("HGetAll").Return(redis.NewStringStringMapCmd("HGetAll"))
	return r
}

func oneSecretRedis() redis.Cmdable {
	r := redismock.NewMock()
	m := make(map[string]string)
	creds := make([]Credential, 0, 1)
	creds = append(creds, Credential{Stage: "current", Value: hmacEncode("hmacKey", "newpassword")})
	m["newuser"] = marshal(creds)
	cmd := redis.NewStringStringMapResult(m, nil)
	r.On("HGetAll").Return(cmd)
	return r
}

func overrideRedis() redis.Cmdable {
	r := redismock.NewMock()
	m := make(map[string]string)
	creds := make([]Credential, 0, 1)
	creds = append(creds, Credential{Stage: "current", Value: hmacEncode("hmacKey", "newpassword")})
	m["user"] = marshal(creds)
	cmd := redis.NewStringStringMapResult(m, nil)
	r.On("HGetAll").Return(cmd)
	return r
}

func marshal(creds []Credential) string {
	b, err := json.Marshal(creds)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func defaultCreds() *BasicAuth {
	ba := NewBasicAuth(metrics.NewRegistry(), "hmacKey")
	ba.AddPrincipal("user", hmacEncode("hmacKey", "password"), "env")
	return ba
}

func newSecretCreds() *BasicAuth {
	ba := defaultCreds()
	ba.AddPrincipal("newuser", hmacEncode("hmacKey", "newpassword"), "current")
	return ba
}

func overrideCreds() *BasicAuth {
	ba := NewBasicAuth(metrics.NewRegistry(), "hmacKey")
	ba.AddPrincipal("user", hmacEncode("hmacKey", "newpassword"), "current")
	return ba
}

func TestRefreshAuth(t *testing.T) {
	tests := map[string]struct {
		auth          *BasicAuth
		client        redis.Cmdable
		config        string
		expectChanged bool
		expectError   error
		expectedCreds *BasicAuth
	}{
		"no secrets in passwordmanager": {
			auth:          defaultCreds(),
			client:        noSecretsRedis(),
			config:        "user:password",
			expectedCreds: defaultCreds(),
			expectChanged: false,
		},
		"new secret added": {
			auth:          defaultCreds(),
			client:        oneSecretRedis(),
			config:        "user:password",
			expectedCreds: newSecretCreds(),
			expectChanged: true,
		},
		"env creds can be overridden": {
			auth:          defaultCreds(),
			client:        overrideRedis(),
			config:        "user:password",
			expectedCreds: overrideCreds(),
			expectChanged: true,
		},
	}

	for name, test := range tests {
		t.Logf("Running test case %s", name)
		changed, err := refreshAuth(test.auth, test.client, "hmacKey", "key", test.config)
		assert.Equal(t, test.expectChanged, changed)
		assert.Equal(t, test.expectError, err)
		assert.Equal(t, test.expectedCreds.creds, test.auth.creds)
	}
}

func TestAuthenticate(t *testing.T) {
	tests := map[string]struct {
		password      string
		authenticated bool
	}{
		"User is authenticated if input password matches": {
			password:      "password",
			authenticated: true,
		},
		"User is not authenticated if input password does not match": {
			password:      "invalidpassword",
			authenticated: false,
		},
	}

	auth, err := NewBasicAuthFromString("user:password", "hmacKey", metrics.NewRegistry())
	if err != nil {
		panic(err.Error())
	}

	for name, test := range tests {
		t.Logf("Running test case %s", name)
		r, err := http.NewRequest("POST", "http://localhost", nil)
		if err != nil {
			panic(err.Error())
		}
		r.SetBasicAuth("user", test.password)
		assert.Equal(t, test.authenticated, auth.Authenticate(r))
	}
}
