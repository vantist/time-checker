## ADDED Requirements

### Requirement: Hook 參數解析安全 Fallback 與預設 Model 載入

當 hook 執行並解析 Prompt 輸入時，系統 SHALL 依據以下邏輯進行參數的 Fallback 與解析：
1. **專案路徑 Fallback**：若 CLI 參數與 stdin payload 均未提供 `project` 路徑，系統 SHALL 使用當前行程之工作目錄 (`os.Getwd()`) 作為專案路徑。
2. **Antigravity 預設 Model 載入**：若整合工具為 `"antigravity"` 且 model 為空字串時，系統 SHALL 主動由 `settings.json`（透過 `GetAntigravityModel`）載入並填入預設的模型名稱。

#### Scenario: 專案路徑 Fallback 至工作目錄
- **WHEN** 呼叫 `tt record prompt` 且未提供 `--project` 且 stdin 無 `cwd` 資訊
- **THEN** 系統呼叫 `os.Getwd()` 取得當前工作目錄並填入 `sessions.project`

#### Scenario: Antigravity 工具無模型名稱時主動載入設定檔預設值
- **WHEN** 呼叫 `tt record prompt --tool antigravity` 且未提供 `--model`，且 `settings.json` 已配置預設模型為 `gemini-2.5-pro`
- **THEN** 系統解析 model 為 `gemini-2.5-pro`
