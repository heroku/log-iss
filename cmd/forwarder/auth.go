package main

import (
	"time"

	"github.com/heroku/authenticater"
)

func newAuth(config AuthConfig) (*authenticater.BasicAuth, error) {
	result, err := authenticater.NewBasicAuthFromString(config.Tokens)
	if err != nil {
		return &result, err
	}

	ticker := time.NewTicker(config.RefreshInterval)
	go func() {
		for _ = range ticker.C {
			refreshAuth(&result, config)
		}
	}()

	return &result, err
}

func refreshAuth(ba *authenticater.BasicAuth, config AuthConfig) error {
	return nil
}
