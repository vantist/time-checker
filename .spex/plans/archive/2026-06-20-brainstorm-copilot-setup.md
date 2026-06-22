# copilot-setup

<!-- Brainstorming plan. Next steps: /spex-propose (full spex flow) or /spex-ingest (add to existing change). For simple plans only: implement directly without spex flow — but NEVER /spex-apply, which requires a prior proposal. -->

## Context

目前 `tt setup --copilot` 僅會印出指示說明，要求使用者手動新增與修改 `~/.copilot/settings.json`。然而，這項指示說明在兩個地方有落差：
1. **設定路徑錯誤**：GitHub Copilot CLI 的使用者級 hooks 設定檔實際應位於 `~/.copilot/hooks/` 目錄下的任何 JSON 檔案（例如 `~/.copilot/hooks/tt.json`）。
2. **JSON 格式錯誤**：Copilot CLI 載入的 hooks 格式並非扁平的 Key-Value，而是帶有 `"version": 1` 且各事件（`userPromptSubmitted`、`agentStop`）對應 `type: "command"` 物件陣列的格式。

同時，其他 AI 工具（Claude Code, Antigravity, Codex）的 setup 都已實現冪等（Idempotent）的自動合併設定，為了保持使用者體驗的一致性與支援的易用性，決定對 Copilot CLI hook 安裝進行自動化與格式修復。

## Decision

將 `tt setup --copilot` 升級為自動化冪等寫入。執行該命令時，`tt` 會自動在 `~/.copilot/hooks/tt.json` 寫入/合併最新的 Copilot hooks（包含 `userPromptSubmitted` 與 `agentStop`），並維持專案原有的 `mergeHooksFile` 冪等邏輯。

## Rationale

1. **使用者體驗一致性**：使用者不需再手動複製 JSON 黏貼至可能錯誤的設定檔路徑，只要執行 `tt setup --copilot` 即可完成。
2. **高魯棒性與安全性**：採用獨立的專用檔案 `tt.json`，在 `mergeHooksFile` 呼叫中過濾 `_owner == "tt"` 項目，在避免破壞使用者或其他工具現有 hook 設定的同時，確保重複執行此命令的安全與乾淨（Idempotency）。

## Approach

1. 在 `internal/setup/setup.go` 中實作 `SetupCopilot() error`，透過 `mergeHooksFile` 自動建立/合併 `~/.copilot/hooks/tt.json`。
2. 更新 `cmd/tt/setup_cmd.go` 中的 `--copilot` 旗標處理邏輯，將原本列印說明的行為替換為執行 `SetupCopilot()`。
3. 於 `internal/setup/setup_test.go` 中撰寫 `TestSetupCopilot` 測試，以模擬 `HOME` 的方式完整測試建檔、內容格式以及冪等合併能力。

## Design Notes

### Go `SetupCopilot` 設計：
```go
func SetupCopilot() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configPath := filepath.Join(home, ".copilot", "hooks", "tt.json")

	updater := func(settings map[string]interface{}) (map[string]interface{}, error) {
		settings["version"] = 1

		hooksSection, _ := settings["hooks"].(map[string]interface{})
		if hooksSection == nil {
			hooksSection = map[string]interface{}{}
		}

		targetHooks := map[string][]interface{}{
			"userPromptSubmitted": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record prompt --tool copilot-cli",
				},
			},
			"agentStop": {
				map[string]interface{}{
					"_owner":  "tt",
					"type":    "command",
					"command": "tt record response --tool copilot-cli",
				},
			},
		}

		for event, newEntries := range targetHooks {
			existing, _ := hooksSection[event].([]interface{})
			var filtered []interface{}
			for _, e := range existing {
				em, _ := e.(map[string]interface{})
				if em["_owner"] != "tt" {
					filtered = append(filtered, e)
				}
			}
			hooksSection[event] = append(filtered, newEntries...)
		}
		
		settings["hooks"] = hooksSection
		return settings, nil
	}

	return mergeHooksFile(configPath, "tt", updater)
}
```

## Insights to Capture

- `design.md`: 新增 Copilot CLI hooks 設定為 `~/.copilot/hooks/tt.json` 自動安裝。
- `specs/idempotent-hook-setup/spec.md`: 補上 Copilot CLI 自動 hooks 設定與冪等合併規格。
- `proposal.md`: 將 Copilot 自動 hooks 設定功能納入 scope。
- `tasks.md`: 新增 SetupCopilot 實作、測試與 CLI 串接之具體 Tasks。

## Open Questions

（無，設計已收斂）
