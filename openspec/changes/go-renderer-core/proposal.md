## Why

bash 版 statusline 在 Windows 上每次 render 需 spawn 一個 MSYS bash 宿主加 8–9 個子程序（jq/git/awk/coreutils），在 `refreshInterval: 1` 的高頻呼叫下，任一子程序無界阻塞即造成殭屍 bash 無限累積、佔住 Cygwin 共享記憶體，最終毒化系統上所有 MSYS 程序（2026-07-12 實際事故）。使用者已因此停用 statusline。需要一個零 MSYS 依賴、程序壽命有硬上界的 renderer，讓狀態列能安全復活。

## What Changes

- 新增 Go 實作的 statusline renderer（單一原生 .exe），作為熱路徑（每次 render 執行）的替代品；bash 版 `statusline.sh` 保留於 repo 但不再註冊使用
- 本批（核心批）涵蓋使用者現行配置所需的 8 個 segment：`ctx`、`git`、`dir`、`model`、`effort`、`limit5h`、`limit7d`、`burn`，以及 pill 風格與固定多行版面（`VL_SEGMENTS` / `VL_SEGMENTS2` / `VL_SEGMENTS3` / `VL_MAX_LINES`）
- 新增 `coralline.conf` 與 `themes/*.conf` 的相容 parser（bash 變數賦值 + 主題 source 指令的子集），設定生態不分裂：同一份 conf、同一批 themes 同時餵 bash 版與 Go 版
- 資料檔格式與現版完全相容：`burn-5h.tsv`（burn 追蹤）、`limit-5h.d` / `limit-7d.d` 快照目錄（跨 session limit sync）、`usage-state.json`（外部 Stop-hook usage guard 依賴此檔）
- 程序安全硬需求：render 程序自帶程序內 watchdog 硬死線（超時自我終止）、stdin 讀取有界、唯一子程序為原生 git.exe 且帶逾時與 `GIT_OPTIONAL_LOCKS=0`、全程零 MSYS 家族程序
- 完成後將 Claude Code 的 statusLine 註冊指向編譯出的 .exe（使用者機器設定，非 repo 檔案）

## Capabilities

### New Capabilities

- `go-renderer-core`: Go renderer 的核心渲染管線——stdin JSON 解析、conf/theme 相容解析、8 個核心 segment、pill 風格與固定多行版面的輸出契約
- `go-renderer-process-safety`: render 程序的壽命與子程序安全保證——watchdog 硬死線、有界 stdin、git 逾時、零 MSYS 程序
- `go-renderer-data-compat`: 與 bash 版共用之持久化資料檔的讀寫相容——burn-5h.tsv、limit 快照目錄、usage-state.json

### Modified Capabilities

(none)

## Impact

- Affected specs: 新增 `go-renderer-core`、`go-renderer-process-safety`、`go-renderer-data-compat`
- Affected code:
  - New: go.mod、cmd/coralline/main.go、internal/ 之下的 Go 套件（conf 解析、segment 渲染、git 狀態、usage 資料）
  - Modified: （無 — configure.sh 與 statusline.sh 本批不動）
  - Removed: （無）
- 系統面：使用者機器需 Go 工具鏈編譯（已安裝 go1.26.5）；Claude Code 使用者設定檔的 statusLine 註冊改指向 .exe（不在 repo 內）
- 後續批次（另立 change）：其餘 8 個 segment、lean/classic 風格與 auto layout
