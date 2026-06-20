## Context

`tt` 時間追蹤器在計算 Token 費用時，若遇到未註冊的 AI 模型，會返回 `nil` (N/A) 成本。隨著使用者整合 `GitHub Copilot CLI`、`Anthropic Claude Code` 與 `Google Antigravity`，許多新模型與版本後綴（如 `-latest`、`-preview`、`-exp`、`-002` 等）在本地日誌中被記錄，導致成本估算錯誤。為了提高計費精準度並防止未來新增變體時失效，需要擴充模型清單並實作穩健的後綴常態化邏輯。

## Goals / Non-Goals

**Goals:**
- 升級 `pricing.normalize` 函數，使用正規表示式動態裁切模型版本、預覽或小版本號等後綴（`-latest`、`-preview`、`-exp`、`-\d{3}`）。
- 擴充 `internal/pricing/pricing.go` 的模型費率對照表 `table`，涵蓋 2026 年最新主流模型定價。
- 在 `internal/pricing/pricing_test.go` 中實作對應的單元測試，驗證常態化與新模型計費邏輯。

**Non-Goals:**
- 不修改資料庫 Schema，也不對資料庫中既有的歷史 Session 進行成本重新計算。
- 不修改 `tt` 計算與寫入估計成本的基本邏輯（遇到未知模型仍保持寫入 NULL 且 exit 0 的行為）。

## Decisions

### 1. 後綴常態化邏輯升級
- **設計**：在 `internal/pricing/pricing.go` 中，更新並擴增正規表示式：
  ```go
  var dateSuffix = regexp.MustCompile(`-\d{8}$`)
  var versionSuffix = regexp.MustCompile(`-(latest|preview|exp|\d{3})$`)
  ```
- **執行順序**：
  1. 移除斜線 `/` 前的 gateway 前綴。
  2. 移除日期後綴（如 `-20260620`）。
  3. 移除版本或預覽後綴（如 `-preview`、`-latest`、`-exp`、`-002`）。
- **理由**：相較於靜態對照所有模型變體，動態裁切後綴能夠自動相容未來釋出的小版本與快照，保證系統的強健性。

### 2. 費率表擴充
- **設計**：在 `table` 中新增/更新 2026 年三大家族（Anthropic Claude 3/3.5/4/5, OpenAI GPT-4o/GPT-5/o1/o3-mini, Google Gemini 1.5/2.5/3/3.5 Pro/Flash/Lite 系列）以及專屬模型（`mai-code-1-flash`, `raptor-mini`, `grok-code-fast-1`）的每百萬 Token 費率。

## Risks / Trade-offs

- **[Risk] 過度裁切模型名稱**：若某些模型名稱本身以 `-001` 等結尾但不是版本號，可能會被誤裁切。
  - **Mitigation**：目前主流 API 模型（如 Claude, Gemini, OpenAI）的 snapshot 命名規範皆符合 `-\d{3}`（三位數字），因此該規則安全度極高。如有特殊模型不適用，可在 `table` 中針對裁切後的 key 進行特別定義，或未來調整正規表示式。
