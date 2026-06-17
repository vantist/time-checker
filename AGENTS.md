# tt — AI Tool Time Tracker

Go CLI，透過 Claude Code / Copilot CLI hook 自動記錄 AI 工作時間與 token 費用。本地 SQLite，單一二進位，零 runtime 依賴。

## Quick Start

```bash
go build -o tt ./cmd/tt   # 建置
tt setup --claude-code    # 安裝 Claude Code hooks
tt report --since 7d      # 查看過去 7 天報表
tt work "feature-xyz"     # 標記目前工作項目
```

## 文件

- [ARCHITECTURE.md](ARCHITECTURE.md) — 模組結構、資料流、DB schema、設計決策
- [docs/commands.md](docs/commands.md) — 完整指令參考、flag 說明、hook 設定範例
- [design.md](design.md) — Hook 整合設計筆記（Claude Code / Copilot CLI stdin 格式）

## 核心慣例

- `internal/` 所有 package 職責單一：db（schema/連線）、recorder（寫入）、report（讀取）、aggregator（時間計算）、pricing（費用）
- Hook 失敗靜默處理（exit 0），不阻擋 AI 工具
- stdin JSON 優先於 CLI flag；flag 保留供測試用
- `TT_DB_PATH` env var 覆寫 DB 路徑（測試隔離用）
