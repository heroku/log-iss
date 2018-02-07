package main

import (
	"testing"
)

func TestIssConfig_LogAuthUser_NoValidUser(t *testing.T) {
	var base = IssConfig{
		TokenUserSamplePct: 100,
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if base.LogAuthUser("user") {
			hits++
		}
	}

	if hits > 0 {
		t.Errorf("expected no hits, but got %d\n", hits)
	}
}

func TestIssConfig_LogAuthUser_ValidUser(t *testing.T) {
	var base = IssConfig{
		ValidTokenUser:     "user",
		TokenUserSamplePct: 100,
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if base.LogAuthUser("user") {
			hits++
		}
	}

	if hits > 0 {
		t.Errorf("expected no hits, but got %d\n", hits)
	}
}

func TestIssConfig_LogAuthUser_ZeroPct(t *testing.T) {
	var base = IssConfig{
		ValidTokenUser:     "user",
		TokenUserSamplePct: 0,
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if base.LogAuthUser("not-user") {
			hits++
		}
	}

	if hits > 0 {
		t.Errorf("expected no hits, but got %d\n", hits)
	}
}

func TestIssConfig_LogAuthUser_OneHundredPct(t *testing.T) {
	var base = IssConfig{
		ValidTokenUser:     "user",
		TokenUserSamplePct: 100,
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if base.LogAuthUser("not-user") {
			hits++
		}
	}

	if hits != 100000 {
		t.Errorf("expected 100000 hits, but got %d\n", hits)
	}
}

func TestIssConfig_LogAuthUser_FiftyPct(t *testing.T) {
	var base = IssConfig{
		ValidTokenUser:     "user",
		TokenUserSamplePct: 50,
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if base.LogAuthUser("not-user") {
			hits++
		}
	}

	if hits < 30000 || hits > 70000 {
		t.Errorf("got %d of 100000, pct calcuation seems off\n", hits)
	}
}
