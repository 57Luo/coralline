## Context

Go binary `cmd/coralline` 已實作 8 個 segment，bash statusline.sh 有 18 個。兩者共用相同的 stdin JSON 格式、相同的 conf 設定檔、相同的 ANSI rendering pipeline（pill/lean/classic layout）。Go binary 的 render pipeline 已完整（`internal/render/`），只需新增 segment 函式和對應的 input 解析。

## Goals / Non-Goals

**Goals:**

- Go binary 支援全部 18 個 segment，與 bash 版 byte-for-byte 輸出相容
- `configure.sh` 安裝時可自動偵測 Go binary 並優先使用

**Non-Goals:**

- 不移除 bash statusline.sh（保留為 fallback）
- 不移植 configure.sh 精靈到 Go
- 不新增任何 bash 版沒有的 segment 或行為

## Decisions

**D1: 新增 `internal/runtime/` package 處理 node/python 偵測**

pin-file walk（.nvmrc, .node-version, .python-version）和 interpreter probe 邏輯獨立於 render，放在專屬 package。函式簽名為 `DetectNode(cwd string, probe bool) string` 和 `DetectPython(cwd string, probe bool) string`，回傳版本字串或空字串。

理由：與 gitstate 平行，各自獨立 package，可獨立測試。

**D2: stash count 擴展 `gitstate.State`**

bash 的 stash segment 執行 `git rev-list --walk-reflogs --count refs/stash`。將此加入 `gitstate.Run()` 的單次 git 呼叫中不可行（需要獨立 command），所以新增 `StashCount` 欄位，在 `gitstate.Run()` 內部做第二次 git 呼叫。

理由：stash count 與 git branch/dirty 狀態在同一 context 下取得，集中管理比 main.go 散落呼叫更乾淨。

**D3: project segment 複用 gitstate.Root**

`gitstate.State` 已有 `Root` 欄位（git repo 根目錄）。project segment 只需 `filepath.Base(git.Root)`，不需額外 git 呼叫。

**D4: Input struct 擴展新欄位**

在 `inputjson.Input` 和 `raw` 新增：Cost (string), LinesAdd (int64), LinesDel (int64), OutStyle (string), DurMs (int64)。JSON path 對應：cost.total_cost_usd, cost.total_lines_added, cost.total_lines_removed, output_style.name, cost.total_duration_ms。

**D5: configure.sh hook 切換**

`update_settings()` 新增 Go binary 偵測：檢查 `$TARGET_DIR/coralline` 或 `$TARGET_DIR/coralline.exe` 是否存在且可執行。若存在，hook command 指向 Go binary；否則 fallback 到 bash statusline.sh。

## Implementation Contract

**Behavior：** 當 Go binary 執行時，所有 18 個 segment 的輸出與 bash 版在相同 input + 相同 conf 下 byte-for-byte 一致。golden test 已有框架（`internal/render/golden_test.go`）— 新 segment 加入同一 test suite。

**新增介面：**

- `runtime.DetectNode(cwd string, probe bool) string` — 回傳 Node 版本字串（不含 `v` prefix）或空字串
- `runtime.DetectPython(cwd string, probe bool) string` — 回傳 Python 環境名稱/版本字串或空字串
- `gitstate.State.StashCount int` — git stash 數量，無 stash 為 0
- `inputjson.Input` 新欄位：`Cost string`, `LinesAdd int64`, `LinesDel int64`, `OutStyle string`, `DurMs int64`

**Failure modes：** 所有新 segment 遵循現有慣例 — 欄位為空/零值時 segment 靜默不輸出。runtime 偵測失敗時回傳空字串。git stash 指令失敗時 StashCount 為 0。

**Acceptance criteria：**
- `go test ./...` 全通過，包含新 segment 的 golden test case
- 以 `test/sample-input.json` 為 input，Go binary 與 bash statusline.sh 輸出完全一致（需將 sample-input.json 擴展新欄位）
- `configure.sh --install` 在 Go binary 存在時，settings.json 的 hook command 指向 Go binary

**Scope boundaries：**
- In scope: 10 個新 segment、input 解析、runtime 偵測、golden test、configure.sh hook 切換
- Out of scope: bash statusline.sh 的任何修改、configure.sh 精靈 UI、Go binary 的 build/release pipeline

## Risks / Trade-offs

- [Risk] stash segment 額外的 git 呼叫增加渲染延遲 → Mitigation: 只在 segment scan 包含 `stash` 時才呼叫，與 bash 行為一致
- [Risk] runtime probe（VL_RUNTIME_PROBE=1）fork `node --version` / `python3 --version` → Mitigation: 預設關閉，與 bash 一致；pin-file walk 零 fork
- [Risk] configure.sh 的 Go binary 偵測在不同平台路徑不同 → Mitigation: 同時檢查 `coralline` 和 `coralline.exe`，覆蓋 Unix/Windows
