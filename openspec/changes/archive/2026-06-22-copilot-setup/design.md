## Context

目前 `tt setup --copilot` 僅會列印指示，要求使用者手動修改設定檔，且設定路徑與格式有誤。我們將為其導入與 Claude Code 等工具類似的自動化、冪等（Idempotent）的 hook 設定寫入與合併邏輯。

## Goals / Non-Goals

**Goals:**
- 自動化寫入與合併 Copilot CLI hook 設定至 `~/.copilot/hooks/tt.json`。
- 實現冪等性，多次執行 `tt setup --copilot` 不會重複添加 hook。
- 不影響使用者自訂的其他 hook 設定。

**Non-Goals:**
- 不支援非 `tt` 所有的 hook 條目之修改或刪除。
- 不支援設定檔路徑的自訂。

## Decisions

### Decision 1: 使用專用設定檔 `~/.copilot/hooks/tt.json`
- **方案 A**: 修改 `~/.copilot/settings.json`（原指示說明所寫）。
- **方案 B**: 寫入 `~/.copilot/hooks/tt.json`。
- **選擇與理由**: 選擇 **方案 B**。因為 GitHub Copilot CLI 的使用者級 hooks 設定檔實際應位於 `~/.copilot/hooks/` 目錄下的 JSON 檔案。使用專屬的 `tt.json` 不僅符合 Copilot CLI 規範，也能避免破壞 `settings.json` 中的其他通用設定。

### Decision 2: 採用 `mergeHooksFile` 進行冪等合併
- **說明**: 沿用現有的 `mergeHooksFile` 通用邏輯，傳入更新器（Updater），在更新器中：
  - 設定 `version = 1`。
  - 初始化/取得 `hooks` 區段。
  - 過濾並移除所有 `_owner == "tt"` 的舊 hooks。
  - 附加新版本的 tt hooks：
    - `userPromptSubmitted`: `tt record prompt --tool copilot-cli`
    - `agentStop`: `tt record response --tool copilot-cli`

## Risks / Trade-offs

- **[Risk]** 目錄 `~/.copilot/hooks/` 不存在。
  - **Mitigation**: `mergeHooksFile` 內部會自動建立父目錄（使用 `os.MkdirAll`），確保目錄存在。
- **[Risk]** 使用者重複執行 `tt setup --copilot` 導致 hook 被重置或重複。
  - **Mitigation**: 通過過濾所有屬主為 `"tt"` 的 hook 再重新寫入，保證寫入的冪等性。
