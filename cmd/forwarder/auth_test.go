package main

import (
	"testing"

	"github.com/elliotchance/redismock"
	"github.com/go-redis/redis"
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
	m["newuser"] = "[\"newpassword\"]"
	cmd := redis.NewStringStringMapResult(m, nil)
	r.On("HGetAll").Return(cmd)
	return r
}

func TestRefreshAuth(t *testing.T) {
	tests := map[string]struct {
		auth          *BasicAuth
		client        redis.Cmdable
		config        string
		expectChanged bool
		expectError   error
		expectedCreds string
	}{
		"no secrets in passwordmanager": {
			auth:          basicAuth(),
			client:        noSecretsRedis(),
			config:        "user:password",
			expectedCreds: "user:password",
			expectChanged: false,
		},
		"new secret added": {
			auth:          basicAuth(),
			client:        oneSecretRedis(),
			config:        "user:password",
			expectedCreds: "user:password|newuser:newpassword",
			expectChanged: true,
		},
	}

	for name, test := range tests {
		t.Logf("Running test case %s", name)
		changed, err := refreshAuth(test.auth, test.client, "key", test.config)
		assert.Equal(t, test.expectChanged, changed)
		assert.Equal(t, test.expectError, err)

		expectedAuth, _ := NewBasicAuthFromString(test.expectedCreds)
		assert.Equal(t, expectedAuth.creds, test.auth.creds)
	}
}
func basicAuth() *BasicAuth {
	result, _ := NewBasicAuthFromString("user:password")
	return result
}
