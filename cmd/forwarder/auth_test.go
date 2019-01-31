package main

import (
	"encoding/json"
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
	creds = append(creds, Credential{Stage: "current", Value: "newpassword"})
	m["newuser"] = marshal(creds)
	cmd := redis.NewStringStringMapResult(m, nil)
	r.On("HGetAll").Return(cmd)
	return r
}

func overrideRedis() redis.Cmdable {
	r := redismock.NewMock()
	m := make(map[string]string)
	creds := make([]Credential, 0, 1)
	creds = append(creds, Credential{Stage: "current", Value: "newpassword"})
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
	ba := NewBasicAuth(metrics.NewRegistry())
	ba.AddPrincipal("user", "password", "env")
	return ba
}

func newSecretCreds() *BasicAuth {
	ba := defaultCreds()
	ba.AddPrincipal("newuser", "newpassword", "current")
	return ba
}

func overrideCreds() *BasicAuth {
	ba := NewBasicAuth(metrics.NewRegistry())
	ba.AddPrincipal("user", "newpassword", "current")
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
		changed, err := refreshAuth(test.auth, test.client, "key", test.config)
		assert.Equal(t, test.expectChanged, changed)
		assert.Equal(t, test.expectError, err)
		assert.Equal(t, test.expectedCreds.creds, test.auth.creds)
	}
}
