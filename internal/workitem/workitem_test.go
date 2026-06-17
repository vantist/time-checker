package workitem_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/tt/internal/workitem"
)

func setup(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
}

func TestSetAndGet(t *testing.T) {
	setup(t)

	if err := workitem.Set("login-redesign"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := workitem.Get()
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "login-redesign" {
		t.Errorf("Get = %q, want %q", got, "login-redesign")
	}

	// Verify file content includes trailing newline
	home, _ := os.UserHomeDir()
	data, _ := os.ReadFile(filepath.Join(home, ".tt", "work-item"))
	if string(data) != "login-redesign\n" {
		t.Errorf("file content = %q, want %q", string(data), "login-redesign\n")
	}
}

func TestGetMissingFileReturnsEmpty(t *testing.T) {
	setup(t)

	got, err := workitem.Get()
	if err != nil {
		t.Fatalf("Get on missing file: %v", err)
	}
	if got != "" {
		t.Errorf("Get = %q, want empty", got)
	}
}

func TestClearDeletesFile(t *testing.T) {
	setup(t)

	_ = workitem.Set("some-task")
	if err := workitem.Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	got, _ := workitem.Get()
	if got != "" {
		t.Errorf("after Clear, Get = %q, want empty", got)
	}
}

func TestClearIdempotent(t *testing.T) {
	setup(t)

	if err := workitem.Clear(); err != nil {
		t.Errorf("Clear on missing file: %v", err)
	}
}
