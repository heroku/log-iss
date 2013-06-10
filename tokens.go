package main

import (
	"errors"
	"strings"
)

type Tokens map[string]string

func ParseTokenMap(tokenMap string) (Tokens, error) {
	tokens := make(Tokens)

	for _, userAndToken := range strings.Split(tokenMap, ",") {
		userAndTokenParts := strings.SplitN(userAndToken, ":", 2)
		if len(userAndTokenParts) != 2 {
			return tokens, errors.New("ENV[TOKEN_MAP] not formatted properly")
		}
		tokens[userAndTokenParts[0]] = userAndTokenParts[1]
	}

	return tokens, nil
}
