package main

import (
	"math/rand"
	"testing"
)

func TestIssConfig_LogAuthUser_NoValidUser(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 100,
		rnd:                rand.New(rand.NewSource(0)),
	}

	var hits int
	for i := 0; i < 100000; i++ {
		if c.LogAuthUser("user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != 0 {
		t.Errorf("expected no hits,  got %d", hits)
	}
}

func TestIssConfig_LogAuthUser_ValidUser(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 100,
		ValidTokenUser:     "user",
		rnd:                rand.New(rand.NewSource(0)),
	}

	var hits int
	for i := 0; i < 100; i++ {
		if c.LogAuthUser("user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != 0 {
		t.Errorf("expected no hits, got %d", hits)
	}
}

func TestIssConfig_LogAuthUser_ZeroPct(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 0,
		ValidTokenUser:     "user",
		rnd:                rand.New(rand.NewSource(0)),
	}

	var hits int
	for i := 0; i < 100; i++ {
		if c.LogAuthUser("not-user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits > 0 {
		t.Errorf("expected no hits, got %d", hits)
	}
}

func TestIssConfig_LogAuthUser_OneHundredPct(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 100,
		ValidTokenUser:     "user",
		rnd:                rand.New(rand.NewSource(0)),
	}

	var hits int
	for i := 0; i < 100; i++ {
		if c.LogAuthUser("not-user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != 100 {
		t.Errorf("expected 100 hits, got %d", hits)
	}
}

func TestIssConfig_LogAuthUser_SeventyFivePct(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 75,
		ValidTokenUser:     "user",
		rnd:                rand.New(rand.NewSource(12)),
	}

	var hits int
	for i := 0; i < 100; i++ {
		if c.LogAuthUser("not-user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != 75 {
		t.Errorf("expected 75 hits, got %d", hits)
	}
}

func TestIssConfig_LogAuthUser_FiftyPct(t *testing.T) {
	c := IssConfig{
		TokenUserSamplePct: 50,
		ValidTokenUser:     "user",
		rnd:                rand.New(rand.NewSource(0)),
	}

	var hits int
	for i := 0; i < 100; i++ {
		if c.LogAuthUser("not-user") {
			hits++
		}
	}

	t.Logf("hits: %d", hits)
	if hits != 50 {
		t.Errorf("expected 50 hits, got %d", hits)
	}
}
