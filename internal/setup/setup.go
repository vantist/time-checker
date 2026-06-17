package setup

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

var ttHooks = map[string]interface{}{
	"UserPromptSubmit": []interface{}{
		map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "tt record prompt"},
			},
		},
	},
	"Stop": []interface{}{
		map[string]interface{}{
			"hooks": []interface{}{
				map[string]interface{}{"type": "command", "command": "tt record response"},
			},
		},
	},
}

func SetupClaudeCode() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		return err
	}
	settingsPath := filepath.Join(claudeDir, "settings.json")

	// Load existing settings
	var settings map[string]interface{}
	data, err := os.ReadFile(settingsPath)
	if errors.Is(err, os.ErrNotExist) {
		settings = map[string]interface{}{}
	} else if err != nil {
		return err
	} else {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = map[string]interface{}{}
		}
	}

	// Merge hooks
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = map[string]interface{}{}
	}
	for event, hook := range ttHooks {
		if _, exists := hooks[event]; !exists {
			hooks[event] = hook
		} else {
			// append to existing list
			existing, _ := hooks[event].([]interface{})
			hooks[event] = append(existing, hook.([]interface{})...)
		}
	}
	settings["hooks"] = hooks

	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath, out, 0o644)
}

const CopilotInstructions = `To set up GitHub Copilot CLI hooks, add the following to ~/.copilot/settings.json:

{
  "hooks": {
    "userPromptSubmitted": "tt record prompt --tool copilot-cli",
    "agentStop": "tt record response --tool copilot-cli"
  }
}

Events:
  userPromptSubmitted  → tt record prompt --tool copilot-cli
  agentStop            → tt record response --tool copilot-cli

Note: Token data is not available in Copilot CLI hooks; token fields will be NULL.
`
