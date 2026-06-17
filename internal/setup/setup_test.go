package setup_test

import (
	"encoding/json"
	"os"
	"path/filepath"
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
