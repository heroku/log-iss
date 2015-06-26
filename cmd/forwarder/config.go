package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/heroku/log-iss/Godeps/_workspace/src/github.com/joeshaw/envdecode"

	metrics "github.com/heroku/log-iss/Godeps/_workspace/src/github.com/rcrowley/go-metrics"
)

type IssConfig struct {
	Deploy                    string        `env:"DEPLOY,required"`
	ForwardDest               string        `env:"FORWARD_DEST,required"`
	ForwardDestConnectTimeout time.Duration `env:"FORWARD_DEST_CONNECT_TIMEOUT,default=10s"`
	ForwardCount              int           `env:"FORWARD_COUNT,default=4"`
	HttpPort                  string        `env:"PORT,required"`
	Tokens                    string        `env:"TOKEN_MAP,required"`
	EnforceSsl                bool          `env:"ENFORCE_SSL,default=false"`
	PemFile                   string        `env:"PEMFILE"`
	LibratoSource             string        `env:"LIBRATO_SOURCE"`
	LibratoOwner              string        `env:"LIBRATO_OWNER"`
	LibratoToken              string        `env:"LIBRATO_TOKEN"`
	Dyno                      string        `env:"DYNO:`
	TlsConfig                 *tls.Config
	MetricsRegistry           metrics.Registry
}

func NewIssConfig() (IssConfig, error) {
	config := IssConfig{}

	err := envdecode.Decode(&config)
	if err != nil {
		return config, err
	}

	if config.PemFile != "" {
		pemFileData, err := ioutil.ReadFile(config.PemFile)
		if err != nil {
			return config, fmt.Errorf("Unable to read pemfile: %s", err)
		}

		cp := x509.NewCertPool()
		if ok := cp.AppendCertsFromPEM(pemFileData); !ok {
			return config, fmt.Errorf("Error parsing PEM: %s", config.PemFile)
		}

		config.TlsConfig = &tls.Config{RootCAs: cp}
	}

	sp := make([]string, 0, 2)
	if config.LibratoSource != "" {
		sp = append(sp, config.LibratoSource)
	}
	if config.Dyno != "" {
		sp = append(sp, config.Dyno)
	}

	config.LibratoSource = strings.Join(sp, ".")

	config.MetricsRegistry = metrics.NewRegistry()

	return config, nil
}
