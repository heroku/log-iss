package main

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

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
	creds = append(creds, Credential{Stage: "current", Hmac: hmacEncode("hmacKey", "newpassword")})
	m["newuser"] = marshal(creds)
	cmd := redis.NewStringStringMapResult(m, nil)
	r.On("HGetAll").Return(cmd)
	return r
}

func overrideRedis() redis.Cmdable {
	r := redismock.NewMock()
	m := make(map[string]string)
	creds := make([]Credential, 0, 1)
	creds = append(creds, Credential{Stage: "current", Hmac: hmacEncode("hmacKey", "newpassword")})
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

func refreshInterval() time.Duration {
	d, err := time.ParseDuration("1m")
	if err != nil {
		panic(err)
	}
	return d
}

func TestNewAuth(t *testing.T) {
	tests := map[string]struct {
		config  AuthConfig
		success bool
	}{
		"Fail if RedisUrl is set but RedisKey is not set": {
			config: AuthConfig{
				RefreshInterval: refreshInterval(),
				RedisUrl:        "redis://localhost:6379/0",
				Tokens:          "user:password",
			},
			success: false,
		},
		"Fail if RedisUrl is unset and Tokens is unset": {
			config:  AuthConfig{},
			success: false,
		},
		"Fail if Token format is invalid": {
			config: AuthConfig{
				RefreshInterval: refreshInterval(),
				RedisUrl:        "redis://localhost:6379/0",
				RedisKey:        "key",
				Tokens:          ":|:",
			},
			success: false,
		},
		"Fail if RedisUrl is not a valid url": {
			config: AuthConfig{
				RefreshInterval: refreshInterval(),
				RedisUrl:        "not-a-real-url",
				RedisKey:        "key",
			},
			success: false,
		},
		"Succeed if Tokens is set properly": {
			config: AuthConfig{
				Tokens: "u1:p1|u2:p2,p3",
			},
			success: true,
		},
		"Succeed if RedisUrl and RedisKey are set properly": {
			config: AuthConfig{
				RefreshInterval: refreshInterval(),
				RedisUrl:        "redis://localhost:6379/0",
				RedisKey:        "key",
			},
			success: true,
		},
	}

	registry := metrics.NewRegistry()

	for name, test := range tests {
		t.Logf("Running test case %s", name)
		_, err := newAuth(test.config, registry)
		if test.success && err != nil {
			assert.Fail(t, err.Error())
		}
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
