## Summary

將 bash statusline.sh 中剩餘 10 個 segment 移植到 Go binary (`cmd/coralline`)，達成完整 segment 覆蓋，使 Claude Code statusline hook 可以完全用 Go 執行檔取代 bash 渲染器。

## Motivation

目前 Go binary 只實作了 8/18 個 segment（dir, git, model, ctx, effort, burn, limit5h, limit7d）。剩餘 10 個 segment 仍需 bash statusline.sh 執行，迫使 Windows 使用者依賴 Git Bash。bash 渲染器每秒被呼叫一次，在 Windows 上會產生 MSYS zombie process 問題。Go binary 已證明能解決這個問題，但 segment 覆蓋不完整使得無法完全切換。

## Proposed Solution

分三層實作缺少的 segment：

**第一層 — 簡單 JSON 格式化（6 個）：** `cost`, `clock`, `lines`, `style`, `duration`, `tokens`。這些只讀取 stdin JSON 欄位，做數值格式化後輸出。需在 `internal/inputjson/inputjson.go` 的 `Input` struct 和 `raw` struct 新增對應欄位（cost.total_cost_usd, cost.total_lines_added, cost.total_lines_removed, output_style.name, cost.total_duration_ms），在 `internal/render/segments.go` 新增 segment 函式。

**第二層 — 環境偵測（2 個）：** `node`, `python`。需在 Go 端實作 pin-file walk（.nvmrc/.node-version/.python-version）和可選的 interpreter probe（VL_RUNTIME_PROBE=1），行為與 bash runtime_node/runtime_python 一致。新增 `internal/runtime/` package。

**第三層 — Git 交互（2 個）：** `stash`（讀 git stash count，需額外一次 git 呼叫）和 `project`（讀 git repo root basename，資訊已在 gitstate 中可取得）。

最後更新 `configure.sh` 的 `update_settings()` 函式，當偵測到已編譯的 Go binary 時，hook command 指向 Go 執行檔而非 bash statusline.sh。

## Non-Goals

- 不移植 `configure.sh` 互動式精靈到 Go（它只在安裝時執行一次）
- 不改變任何 segment 的視覺輸出或行為（byte-for-byte 相容是目標）
- 不新增 bash 版沒有的 segment
- 不改變 Go binary 的 watchdog timeout 或 stdin 讀取機制

## Impact

- Affected code:
  - Modified: `internal/inputjson/inputjson.go`, `internal/inputjson/inputjson_test.go`, `internal/render/segments.go`, `internal/render/segments_test.go`, `internal/render/render.go`, `internal/gitstate/gitstate.go`, `configure.sh`, `cmd/coralline/main.go`
  - New: `internal/runtime/runtime.go`, `internal/runtime/runtime_test.go`
