package setup_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/tt/internal/setup"
)

func setupHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

// Task 8.1: SetupClaudeCode writes hooks when settings.json absent
func TestSetupClaudeCodeFresh(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings.json not created: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		t.Fatal("hooks key missing or wrong type")
	}
	for _, event := range []string{"UserPromptSubmit", "Stop"} {
		if _, ok := hooks[event]; !ok {
			t.Errorf("hooks.%s missing", event)
		}
	}
}

// TestSetupClaudeCode_HookCommand: UserPromptSubmit hook command contains PROCESS_PID env var.
func TestSetupClaudeCode_HookCommand(t *testing.T) {
	home := setupHome(t)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings.json: %v", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	hooks := settings["hooks"].(map[string]interface{})
	entries := hooks["UserPromptSubmit"].([]interface{})
	var cmd string
	for _, e := range entries {
		em := e.(map[string]interface{})
		hs := em["hooks"].([]interface{})
		for _, h := range hs {
			hm := h.(map[string]interface{})
			if c, ok := hm["command"].(string); ok {
				cmd = c
				break
			}
		}
	}

	if cmd == "" {
		t.Fatal("UserPromptSubmit hook command is empty")
	}
	if !strings.Contains(cmd, "PROCESS_PID") {
		t.Errorf("hook command %q does not contain PROCESS_PID", cmd)
	}
}

// Task 8.1: existing hooks not overwritten
func TestSetupClaudeCodePreservesExistingHooks(t *testing.T) {
	home := setupHome(t)

	// Pre-populate settings with an existing hook
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0o755)
	existing := map[string]interface{}{
		"hooks": map[string]interface{}{
			"PreToolUse": []interface{}{
				map[string]interface{}{"hooks": []interface{}{map[string]interface{}{"type": "command", "command": "caveman-hook"}}},
			},
		},
	}
	data, _ := json.Marshal(existing)
	os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	if err := setup.SetupClaudeCode(); err != nil {
		t.Fatalf("SetupClaudeCode: %v", err)
	}

	data, _ = os.ReadFile(filepath.Join(claudeDir, "settings.json"))
	var settings map[string]interface{}
	json.Unmarshal(data, &settings)

	hooks := settings["hooks"].(map[string]interface{})
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Error("existing PreToolUse hook was removed")
	}
	if _, ok := hooks["UserPromptSubmit"]; !ok {
		t.Error("tt UserPromptSubmit hook not added")
	}
}
