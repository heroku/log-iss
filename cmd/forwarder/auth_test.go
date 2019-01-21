package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	sm "github.com/aws/aws-sdk-go/service/secretsmanager"
	smi "github.com/aws/aws-sdk-go/service/secretsmanager/secretsmanageriface"
	"github.com/stretchr/testify/assert"
)

type mockedSMI struct {
	smi.SecretsManagerAPI
	Lso              sm.ListSecretsOutput
	ExpectedSecretId string
	Gsvo             sm.GetSecretValueOutput
}

func (m mockedSMI) ListSecretsPages(input *sm.ListSecretsInput, fn func(*sm.ListSecretsOutput, bool) bool) error {
	_ = fn(&m.Lso, true)
	return nil
}

func (m mockedSMI) GetSecretValue(input *sm.GetSecretValueInput) (*sm.GetSecretValueOutput, error) {
	return &m.Gsvo, nil
}

func TestRefreshAuth(t *testing.T) {
	tests := map[string]struct {
		auth          *BasicAuth
		client        smi.SecretsManagerAPI
		prefix        string
		config        string
		expectChanged bool
		expectError   error
		expectedCreds string
	}{
		"no secrets in passwordmanager": {
			auth: basicAuth(),
			client: mockedSMI{
				Lso: sm.ListSecretsOutput{
					SecretList: []*sm.SecretListEntry{},
				},
				Gsvo: sm.GetSecretValueOutput{},
			},
			prefix:        "log-iss",
			config:        "user:password",
			expectedCreds: "user:password",
		},
		"new secret added": {
			auth: basicAuth(),
			client: mockedSMI{
				Lso: sm.ListSecretsOutput{
					SecretList: []*sm.SecretListEntry{
						&sm.SecretListEntry{
							ARN:  aws.String("secret-arn"),
							Name: aws.String("log-iss/newuser"),
						},
					},
				},
				Gsvo: sm.GetSecretValueOutput{
					ARN:          aws.String("secret-arn"),
					Name:         aws.String("log-iss/newuser"),
					SecretString: aws.String("newpassword"),
				},
			},
			prefix:        "log-iss",
			config:        "user:password",
			expectedCreds: "user:password|newuser:newpassword",
			expectChanged: true,
		},
	}

	for name, test := range tests {
		t.Logf("Running test case %s", name)
		changed, err := refreshAuth(test.auth, test.client, test.prefix, test.config)
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
