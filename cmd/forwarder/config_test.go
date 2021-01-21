package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryFieldParams(t *testing.T) {
	assert := assert.New(t)

	setupDefaultEnv()
	os.Setenv("LOG_ISS_FIELD_PARAMS", "custom1;custom2;custom3")

	config, err := NewIssConfig()
	if err != nil {
		assert.FailNow(err.Error())
	}

	assert.ElementsMatch([]string{"custom1", "custom2", "custom3"}, config.QueryFieldParams)
}

func setupDefaultEnv() {
	os.Setenv("DEPLOY", "codetest")
	os.Setenv("FORWARD_DEST", "127.0.0.1:5001")
	os.Setenv("PORT", "8080")
}
