package main

import (
	"testing"
	"time"
)

func TestDurationFromEnv(t *testing.T) {
	t.Setenv("TEST_DELAY", "3s")

	got, err := durationFromEnv("TEST_DELAY", 0)
	if err != nil {
		t.Fatalf("durationFromEnv() unexpected error: %v", err)
	}
	if got != 3*time.Second {
		t.Errorf("durationFromEnv() = %v, want %v", got, 3*time.Second)
	}
}

func TestDurationFromEnvUsesFallback(t *testing.T) {
	t.Setenv("TEST_DELAY", "")

	want := time.Second
	got, err := durationFromEnv("TEST_DELAY", want)
	if err != nil {
		t.Fatalf("durationFromEnv() unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("durationFromEnv() = %v, want %v", got, want)
	}
}

func TestDurationFromEnvRejectsInvalidValue(t *testing.T) {
	t.Setenv("TEST_DELAY", "slow")

	if _, err := durationFromEnv("TEST_DELAY", 0); err == nil {
		t.Fatal("durationFromEnv() error = nil, want parsing error")
	}
}

func TestDurationFromEnvRejectsNegativeValue(t *testing.T) {
	t.Setenv("TEST_DELAY", "-1s")

	if _, err := durationFromEnv("TEST_DELAY", 0); err == nil {
		t.Fatal("durationFromEnv() error = nil, want negative-duration error")
	}
}
