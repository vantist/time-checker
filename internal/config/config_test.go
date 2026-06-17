package config_test

import (
	"testing"

	"github.com/user/tt/internal/config"
)

func setup(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
}

func TestSetAndGet(t *testing.T) {
	setup(t)

	if err := config.Set("idle-threshold", "30"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := config.Get("idle-threshold")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "30" {
		t.Errorf("Get = %q, want %q", got, "30")
	}
}

func TestGetDefaultIdleThreshold(t *testing.T) {
	setup(t)

	got, err := config.Get("idle-threshold")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "15" {
		t.Errorf("default idle-threshold = %q, want %q", got, "15")
	}
}

func TestGetUnknownKeyEmpty(t *testing.T) {
	setup(t)

	got, err := config.Get("nonexistent-key")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "" {
		t.Errorf("unknown key = %q, want empty", got)
	}
}
