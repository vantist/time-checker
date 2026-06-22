## ADDED Requirements

### Requirement: 自動修補歷史 Session 的分支與中繼資料

系統在執行 `reconcile` 前，SHALL 自動修復歷史 Session 中缺失的專案路徑 (`project`)、模型名稱 (`model`) 與 Git 分支名稱 (`branch`)。

#### Scenario: 歷史 Session 的 Git 分支修復成功
- **WHEN** Session 的 `branch` 為空（或為空字串），且專案路徑 `project` 為有效的 Git 專案目錄
- **THEN** 系統自動解析並修復其 `branch` 欄位為該專案目前之 Git 分支名稱

#### Scenario: 非 Git 專案歷史 Session 的分支標記為佔位符
- **WHEN** Session 的 `branch` 為空，且專案路徑 `project` 為非 Git 目錄（或解析失敗）
- **THEN** 系統自動修復並標記該 `branch` 欄位為 "-"，以防止後續重複解析
