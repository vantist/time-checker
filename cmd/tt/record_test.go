package main

import (
	"testing"
)

// TestResolvePromptInput_EnvVars: PROCESS_PID and PROCESS_START env vars are
// parsed and placed into PromptInput correctly.
func TestResolvePromptInput_EnvVars(t *testing.T) {
	t.Setenv("PROCESS_PID", "12345")
	t.Setenv("PROCESS_START", "1700000000")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("resolvePromptInputFromEnv: %v", err)
	}

	if input.ProcessPID != 12345 {
		t.Errorf("ProcessPID = %d, want 12345", input.ProcessPID)
	}
	if input.ProcessStart != 1700000000 {
		t.Errorf("ProcessStart = %d, want 1700000000", input.ProcessStart)
	}
}

// TestResolvePromptInput_EnvVars_InvalidStart: invalid PROCESS_START → ProcessStart=0.
func TestResolvePromptInput_EnvVars_InvalidStart(t *testing.T) {
	t.Setenv("PROCESS_PID", "12345")
	t.Setenv("PROCESS_START", "notanumber")

	input, err := resolvePromptInputFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if input.ProcessStart != 0 {
		t.Errorf("ProcessStart = %d, want 0 (degraded)", input.ProcessStart)
	}
}
