package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/joeshaw/envdecode"

	"github.com/heroku/go-metrics"
)

type IssConfig struct {
	Deploy                    string        `env:"DEPLOY,required"`
	ForwardDest               string        `env:"FORWARD_DEST,required"`
	ForwardDestConnectTimeout time.Duration `env:"FORWARD_DEST_CONNECT_TIMEOUT,default=10s"`
	ForwardCount              int           `env:"FORWARD_COUNT,default=4"`
	HttpPort                  string        `env:"PORT,required"`
	EnforceSsl                bool          `env:"ENFORCE_SSL,default=false"`
	PemFile                   string        `env:"PEMFILE"`
	LibratoSource             string        `env:"LIBRATO_SOURCE"`
	LibratoOwner              string        `env:"LIBRATO_OWNER"`
	LibratoToken              string        `env:"LIBRATO_TOKEN"`
	Dyno                      string        `env:"DYNO"`
	MetadataId                string        `env:"METADATA_ID"`
	Debug                     bool          `env:"LOG_ISS_DEBUG"`
	QueryFieldParams          []string      `env:"LOG_ISS_FIELD_PARAMS"`
	QueryParams               []string      `env:"LOG_ISS_QUERY_PARAMS"`
	TlsConfig                 *tls.Config
	MetricsRegistry           metrics.Registry
}

type AuthConfig struct {
	HmacKey         string        `env:"HMAC_KEY,required"`
	RedisUrl        string        `env:"REDIS_URL"`
	RedisKey        string        `env:"REDIS_KEY"`
	RefreshInterval time.Duration `env:"CREDENTIAL_REFRESH_INTERVAL,default=1m,strict"`
	Tokens          string        `env:"TOKEN_MAP"`
}

func NewAuthConfig() (AuthConfig, error) {
	var config AuthConfig
	err := envdecode.Decode(&config)
	return config, err
}

func NewIssConfig() (IssConfig, error) {
	var config IssConfig
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
