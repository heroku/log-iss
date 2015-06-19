package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"
)

type IssConfig struct {
	Deploy                    string
	ForwardDest               string
	ForwardDestConnectTimeout time.Duration
	HttpPort                  string
	Tokens                    string
	EnforceSsl                bool
	TlsConfig                 *tls.Config
}

func NewIssConfig() (IssConfig, error) {
	config := IssConfig{}

	deploy, err := MustEnv("DEPLOY")
	if err != nil {
		return config, err
	}
	config.Deploy = deploy

	forwardDest, err := MustEnv("FORWARD_DEST")
	if err != nil {
		return config, err
	}
	config.ForwardDest = forwardDest

	var forwardDestConnectTimeout int
	forwardDestConnectTimeoutEnv := os.Getenv("FORWARD_DEST_CONNECT_TIMEOUT")
	if forwardDestConnectTimeoutEnv != "" {
		forwardDestConnectTimeout, err = strconv.Atoi(forwardDestConnectTimeoutEnv)
		if err != nil {
			return config, fmt.Errorf("Unable to parse FORWARD_DEST_CONNECT_TIMEOUT: %s", err)
		}
	} else {
		forwardDestConnectTimeout = 10
	}
	config.ForwardDestConnectTimeout = time.Duration(forwardDestConnectTimeout) * time.Second

	httpPort, err := MustEnv("PORT")
	if err != nil {
		return config, err
	}
	config.HttpPort = httpPort

	config.Tokens, err = MustEnv("TOKEN_MAP")
	if err != nil {
		return config, err
	}

	if os.Getenv("ENFORCE_SSL") == "1" {
		config.EnforceSsl = true
	}

	if pemFile := os.Getenv("PEMFILE"); pemFile != "" {
		pemFileData, err := ioutil.ReadFile(pemFile)
		if err != nil {
			return config, fmt.Errorf("Unable to read pemfile: %s", err)
		}

		cp := x509.NewCertPool()
		if ok := cp.AppendCertsFromPEM(pemFileData); !ok {
			return config, fmt.Errorf("Error parsing PEM: %s", pemFile)
		}

		config.TlsConfig = &tls.Config{RootCAs: cp}
	}
	return config, nil
}

func MustEnv(key string) (string, error) {
	value := os.Getenv(key)
	if value == "" {
		return "", fmt.Errorf("ENV[%s] is required", key)
	}
	return value, nil
}
