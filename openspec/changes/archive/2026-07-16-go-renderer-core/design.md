## Context

coralline 的熱路徑 `statusline.sh` 由 Claude Code 以 `refreshInterval: 1` 高頻呼叫，Windows 上每次 render 產生一個 MSYS bash 宿主與 8–9 個子程序。MSYS/Cygwin 全家族共用一塊共享記憶體，任一程序無界阻塞即毒化整個家族（2026-07-12 事故），使用者已停用 statusline。本設計以 Go 重寫熱路徑為單一原生執行檔；冷路徑（configure.sh、install.sh、themes）與設定生態（`coralline.conf`、`themes/*.conf`）維持不變。使用者現行配置：pill 風格、catppuccin-mocha 主題、三行固定版面、segment 為 ctx/git/dir/model/effort/limit5h/limit7d/burn、`VL_LIMIT_SYNC=1`、`VL_USAGE_STATE=1`（外部 Stop-hook usage guard 讀取 usage-state.json）。

## Goals / Non-Goals

**Goals:**

- render 熱路徑零 MSYS 家族程序；唯一子程序為原生 git.exe 且帶逾時
- 每個 render 程序自帶程序內硬死線（watchdog），任何情況下壽命有上界
- 使用者現行 8 個 segment 在 pill 風格 + 固定多行版面下輸出與 bash 版視覺等價
- 讀同一份 `coralline.conf` 與 `themes/*.conf`，設定生態不分裂
- `usage-state.json`、`burn-5h.tsv`、`limit-5h.d` / `limit-7d.d` 讀寫格式與 bash 版逐位元組相容（可與 bash 版交替執行不互相破壞）

**Non-Goals:**

- lean / classic 風格與 `VL_LAYOUT=auto` 自動換行（後續風格批）
- 其餘 8 個 segment：project/node/python/lines/cost/style/duration/stash/clock/tokens（後續補齊批）
- configure.sh 精靈改造或自動註冊 .exe（本批註冊為手動編輯 settings.json）
- 回饋 upstream、跨平台支援（本 fork 明確 Windows-only）
- 不修改、不刪除 `statusline.sh`（保留為行為對照的參考實作）

## Decisions

### 選擇 Go 單一執行檔而非 Node 或 PowerShell

啟動成本 ~10ms（Node ~50–100ms、pwsh 數百 ms），此稅每秒繳一次。更關鍵的是 watchdog 可靠度：Go 以獨立 goroutine 計時 + `os.Exit`，不論主邏輯卡在哪個 syscall 都無條件開槍；Node 的 `setTimeout` 在 event loop 被同步呼叫凍結時失效。零執行期依賴（編譯後不需任何 runtime）。已於使用者機器安裝 go1.26.5。

### 模組佈局：go.mod 置於 repo 根目錄，程式碼收在 cmd 與 internal

`go.mod`（module 名 `coralline`）+ `cmd/coralline/main.go` + `internal/` 套件：`conf`（設定解析）、`inputjson`（stdin JSON 契約）、`gitstate`（git 子程序）、`usage`（limit 快照、burn、usage-state）、`render`（ANSI、bar、pill 組裝、版面）。bash 專案加入 Go 原始碼互不干擾；建置產物 coralline.exe 不入版控（.gitignore）。

### conf 相容解析器只支援生成子集，不模擬 bash

`coralline.conf` 與 themes 由 configure.sh 生成，實際只用到三種構文：`VAR=value`、`VAR="value"`（含 `${VAR:-default}` 不出現於生成檔）、以及主題引入行 `. "$_VL_DIR/themes/<name>.conf"`。解析器支援：註解與空行略過、上述賦值（單雙引號與裸值）、source 行展開為就地解析該主題檔（`$_VL_DIR` 解析為 renderer 執行檔所在目錄——對齊 bash 版 `_VL_DIR="${0%/*}"`＝statusline.sh 自身目錄，真實部署為 `~/.claude/coralline`，在 conf 目錄 `~/.claude` 的下一層；2026-07-14 verifier 以真實 conf 實測抓到原先「conf 所在目錄」語意會靜默漏掉主題）。不在子集內的行一律靜默略過（與 bash 版對未知變數的行為等價：不影響輸出）。優先序：內建預設 → conf（含 source 的主題）逐行覆寫，與 bash source 語意一致。

### watchdog 以 goroutine 硬死線實作，逾時靜默退出

main 起始即 `time.AfterFunc(5s, func(){ os.Exit(1) })`。逾時不輸出任何內容（Claude Code 顯示上一次成功的狀態列），exit code 1。5 秒與 bash 版 `read -t 5` 的既有校準一致，不另設環境變數。stdin 讀取另設 4MB 上限防止異常輸入撐爆記憶體。

### git 子程序帶 context 逾時與 GIT_OPTIONAL_LOCKS=0

以 `exec.CommandContext`（2.5 秒逾時）執行 `git -C <cwd> status --porcelain=v2 --branch`，env 帶 `GIT_OPTIONAL_LOCKS=0`；逾時或失敗時 git segment 隱藏（等價 bash 版 `2>/dev/null` 後空輸出的行為）。逾時後明確 Kill 子程序，不留孤兒。本批不啟用 project segment，故不需 rev-parse。

### 資料檔演算法自 bash 參考實作逐一移植，格式逐位元組相容

- `usage-state.json`：單行 JSON，欄位與序列化順序照 bash 版 printf 模板（source/updated_at/model/five_hour/seven_day），temp+rename 原子寫入，孤兒 tmp 清掃。
- `limit-5h.d` / `limit-7d.d`：快照為空目錄，目錄名 `<%010d reset-epoch>_<%07.3f pct>`；讀取取 reset 最大者、GC 其餘、修剪超出 now+視窗上限的中毒哨兵（沿用 bash 版自癒邏輯）。
- `burn-5h.tsv`：`epoch\tpct\treset_epoch` 三欄 TSV；ETA 斜率擬合（僅取當前視窗、跨越點數與最小時距守門、warming/idle/active 三態）與修剪邏輯照 awk 參考實作移植為 Go。
- 相容的定義：Go 版寫出的檔案 bash 版可讀且語意不變，反之亦然；兩版交替執行不損壞資料。

### 以 bash 版為 oracle 的視覺等價驗證

開發期以同一份 stdin JSON 分別餵 bash 版與 Go 版，比對輸出（容許差異僅限行尾空白與逾時性欄位如時鐘）；另備 `testdata/` 固定輸入的 golden test 供迴歸。單元測試覆蓋 conf 解析、fmt_tok 縮寫（1234→1.2k）、bar 填充四捨五入、limit 快照 GC、burn 三態。

## Implementation Contract

**行為**：`coralline.exe` 從 stdin 讀入 Claude Code session JSON，向 stdout 輸出至多 `VL_MAX_LINES` 行帶 ANSI 色彩的 pill 風格狀態列後退出。任何錯誤（JSON 壞損、conf 缺失、git 失敗）都以「隱藏對應 segment / 使用內建預設」降級，不 panic、不輸出錯誤訊息到 stdout。

**輸入契約**（stdin JSON，取用欄位）：`workspace.current_dir`（後備 `cwd`）、`model.display_name`、`context_window.used_percentage`、`context_window.total_input_tokens`、`context_window.total_output_tokens`、`context_window.current_usage.cache_read_input_tokens`、`context_window.current_usage.cache_creation_input_tokens`、`rate_limits.five_hour.used_percentage`、`rate_limits.five_hour.resets_at`、`rate_limits.seven_day.used_percentage`、`rate_limits.seven_day.resets_at`、`effort.level`。缺欄位一律以空值/零值處理。

**設定契約**：讀 `~/.claude/coralline.conf`（可由 `VL_CONFIG` 環境變數覆寫路徑，與 bash 版一致），支援本批 segment 用到的 VL_* 變數（含 VL_STYLE、VL_SEGMENTS、VL_SEGMENTS2、VL_SEGMENTS3、VL_MAX_LINES、VL_COLS、VL_CTX_TOKENS、VL_WARN_PCT、VL_HOT_PCT、VL_LIMIT_SYNC、VL_USAGE_STATE、VL_CLOCK、bar 與主題色變數）。

**失敗模式**：watchdog 逾時 → exit 1、無輸出；stdin 超過 4MB → 截斷於 4MB 後照常解析（解析失敗則各欄位空值降級）；git 逾時 → git segment 隱藏、子程序被 Kill。

**驗收標準**：
- `go build ./...` 與 `go test ./...` 全綠（不得 skip）
- 以 `~/.claude/coralline/sample-input.json` 餵入，Go 版與 bash 版輸出視覺等價（golden test + 人工目視）
- 殭屍驗證：以永不 EOF 的 stdin 啟動（例如管線另一端保持開啟），程序在 5 秒內自行退出
- 程序驗證：render 期間工作管理員/Get-CimInstance 觀測不到任何 bash.exe/awk/coreutils，僅有 coralline.exe 與至多一個 git.exe
- 相容驗證：Go 版寫入 usage-state.json / burn-5h.tsv / limit 快照後，手動執行 bash 版可正常讀取並續寫，反之亦然

**範圍邊界**：in scope = 上述 8 個 segment、pill 風格、固定版面、三個資料檔、watchdog 與 git 安全；out of scope = 其餘 segment、lean/classic、auto layout、configure.sh、statusline.sh 的任何修改。

## Risks / Trade-offs

- [視覺等價的長尾：pill 分隔、色彩過渡、Nerd Font 字寬等細節在兩實作間出現細微差異] → 以 bash 版為 oracle 做同輸入比對，差異逐項收斂；驗收含人工目視
- [burn ETA 的 awk 演算法含多個守門與自癒細節，移植時漏抄造成 ETA 失真] → 逐段對照 awk 參考實作移植並為 warming/idle/active 三態與哨兵修剪各寫單元測試
- [兩版交替執行時資料檔競態] → 沿用 bash 版既有策略：原子 temp+rename、快照為 rmdir 安全的空目錄、孤兒 tmp 清掃；相容驗證含交替執行情境
- [Go 工具鏈在公司機器的政策風險] → 已以 winget 安裝 go1.26.5 成功；編譯僅開發期需要，執行檔零依賴
- [後續批次前，未移植的 segment 若被使用者加入 conf，Go 版靜默不顯示] → 可接受：與 bash 版對未知 segment 名的行為一致（略過），後續批補齊

## Migration Plan

1. 建置 coralline.exe 並複製至 `~/.claude/coralline/`
2. 於 Claude Code 使用者 settings.json 加回 statusLine 註冊，command 指向 coralline.exe（目前為停用狀態，無需先拆 bash 註冊）
3. 回滾：把 statusLine 註冊改回 bash 命令即可（statusline.sh 未動）

## Open Questions

（無 — 語言選型、批次切分、資料檔相容範圍均已與使用者確認）
