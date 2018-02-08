package main

import (
	"testing"
)

func configTest(t *testing.T, c IssConfig, user string, expected int) {
	var hits int
	for i := 1; i <= 100; i++ {
		if c.LogAuthUser(user, i) {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != expected {
		t.Errorf("expected %d hits, got %d", expected, hits)
	}
}

func TestIssConfig_LogAuthUser_NoValidUser(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 100,
		},
		"user",
		0,
	)
}

func TestIssConfig_LogAuthUser_ValidUser(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 100,
			ValidTokenUser:     "user",
		},
		"user",
		0,
	)
}

func TestIssConfig_LogAuthUser_ZeroPct(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 0,
			ValidTokenUser:     "user",
		},
		"not-user",
		0,
	)
}

func TestIssConfig_LogAuthUser_OneHundredPct(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 100,
			ValidTokenUser:     "user",
		},
		"not-user",
		100,
	)
}

func TestIssConfig_LogAuthUser_SeventyFivePct(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 75,
			ValidTokenUser:     "user",
		},
		"not-user",
		75,
	)
}

func TestIssConfig_LogAuthUser_FiftyPct(t *testing.T) {
	configTest(t,
		IssConfig{
			TokenUserSamplePct: 50,
			ValidTokenUser:     "user",
		},
		"not-user",
		50,
	)
}
