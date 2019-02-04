package main

type Credential struct {
	Stage string `json:stage`
	Hmac  string `json:hmac`
}
