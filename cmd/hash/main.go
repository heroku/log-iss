package main

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"os"

	"github.com/joeshaw/envdecode"
)

type AuthConfig struct {
	HmacKey string `env:"HMAC_KEY"`
}

func hmacEncode(key string, value string) string {
	secret := []byte(key)
	message := []byte(value)
	hash := hmac.New(sha512.New, secret)
	hash.Write(message)
	return hex.EncodeToString(hash.Sum(nil))
}

func main() {
	var config AuthConfig
	err := envdecode.Decode(&config)
	if err != nil {
		log.Fatalln(err)
	}

	password := os.Args[1]
	fmt.Printf(hmacEncode(config.HmacKey, password))
}
