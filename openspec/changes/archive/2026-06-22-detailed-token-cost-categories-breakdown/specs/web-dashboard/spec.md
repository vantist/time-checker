## MODIFIED Requirements

### Requirement: 網頁 dashboard 顯示 By Project table

dashboard SHALL 包含 By Project table，欄位為 project、sessions、agent time、user time、tokens、cost。Tokens 欄位應支援懸停顯示 4 類 token 細目。

#### Scenario: By Project table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且有多個 project 的 sessions
- **THEN** table 每列對應一個 project，包含 project 名稱、session 數、agent time、user time、以及 tokens 欄位、est. cost
- **THEN** tokens 欄位顯示 Input + Output 總量（或格式化顯示），並支援 hover 懸停顯示 Input, Output, Cache read, Cache creation 的明細提示框

### Requirement: 網頁 dashboard 顯示 By Agent table

dashboard SHALL 包含 By Agent table，欄位為 agent、sessions、agent time、user time、tokens、cost。Tokens 欄位應支援懸停顯示 4 類 token 細目。

#### Scenario: By Agent table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且有多個 agent 的 sessions
- **THEN** table 每列對應一個 agent，包含正規化後的 agent 名稱、session 數、agent time、user time、以及 tokens 欄位、est. cost
- **THEN** tokens 欄位顯示 Input + Output 總量，並支援 hover 懸停顯示 Input, Output, Cache read, Cache creation 的明細提示框

### Requirement: 網頁 dashboard 顯示 Session 明細 table

dashboard SHALL 包含 Session 明細 table，每列對應一筆 session，欄位含時間、project、branch、agent、model、turns、agent time、user time、work item、tokens、cost。

#### Scenario: Session 明細 table 渲染

- **WHEN** 瀏覽器請求 `GET /` 且 DB 有 sessions
- **THEN** table 每列包含 session 開始時間（local time）、project、branch、正規化後的 agent 名稱、model、turns 數、agent time、user time、work item、tokens 欄位、est. cost
- **THEN** tokens 欄位顯示該 session 的總 token 數，並支援 hover 懸停顯示 4 類 token 細目

### Requirement: By Work Item table 顯示 Project 欄

dashboard 的 By Work Item table SHALL 包含 Project 欄，顯示 `GroupResult.Project`（即 `path.Base(project)`），位於 Label 欄右側，且包含 `Tokens` 欄位，支援懸停顯示詳細 Token 類別。

#### Scenario: By Work Item table 包含 Project 欄

- **WHEN** 瀏覽器請求 `GET /` 且 BY WORK ITEM 報表有資料
- **THEN** By Work Item table thead 包含 `Project` 與 `Tokens` 欄位標題
- **THEN** 每列對應的 `<td>` 顯示該 group 的 `path.Base(project)` 值
- **THEN** Tokens 欄位顯示總 Token 數，懸停時顯示 Input, Output, Cache read, Cache creation 詳細分類

#### Scenario: 相同 label 不同 project 顯示為獨立列

- **WHEN** 兩個 repo 均有 branch = "main" 的 sessions
- **THEN** By Work Item table 顯示兩列，Label 欄均為 "main"，Project 欄各自顯示不同的 repo 名稱

### Requirement: 網頁 dashboard 顯示 By Model & Role 佔比與明細

網頁 dashboard (`GET /`) SHALL 包含「By Model & Role」統計區塊，以圖表或比例條呈現不同 Model 與不同角色的 Token 消耗與費用佔比，並以表格列出各 Model/Role 的明細（包含 Model, Role, Tokens, Cost，其中 Tokens 欄位支援懸停顯示細目）。

#### Scenario: Dashboard 顯示 By Model & Role 明細

- **WHEN** 瀏覽器請求 `GET /` 且資料庫有包含 subagent 的 token 記錄
- **THEN** 頁面包含 Model 與 Role 的百分比統計條
- **THEN** 頁面包含明細表格，列出各 Model 在不同角色 (Main/Subagent) 下的 Token 與費用
- **THEN** 表格中的 Tokens 欄位顯示 Input + Output 總量，懸停時顯示 4 類 token 細目
