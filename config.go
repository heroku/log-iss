package main

import (
	"fmt"
	"os"
)

type IssConfig struct {
	Deploy      string
	ForwardDest string
	HttpPort    string
	Tokens      Tokens
}

func NewIssConfig() (*IssConfig, error) {
	config := new(IssConfig)

	deploy, err := MustEnv("DEPLOY")
	if err != nil {
		return nil, err
	}
	config.Deploy = deploy

	forwardDest, err := MustEnv("FORWARD_DEST")
	if err != nil {
		return nil, err
	}
	config.ForwardDest = forwardDest

	httpPort, err := MustEnv("PORT")
	if err != nil {
		return nil, err
	}
	config.HttpPort = httpPort

	tokenMap, err := MustEnv("TOKEN_MAP")
	if err != nil {
		return nil, err
	}
	tokens, err := ParseTokenMap(tokenMap)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse tokens: %s", err)
	}
	config.Tokens = tokens

	return config, nil
}

func MustEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("ENV[%s] is required", key)
	}
	return value, nil
}
