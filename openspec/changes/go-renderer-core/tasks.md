## 1. 模組骨架

- [x] 1.1 依 design 決策「選擇 Go 單一執行檔而非 Node 或 PowerShell」與「模組佈局：go.mod 置於 repo 根目錄，程式碼收在 cmd 與 internal」建立 go.mod（module 名 coralline）、cmd/coralline/main.go 與 internal/{conf,inputjson,gitstate,usage,render} 套件骨架，並在 .gitignore 排除 coralline.exe；驗證：`go build ./...` 成功產出可執行檔且 `git status` 不出現建置產物

## 2. 設定解析（Configuration file compatibility）

- [x] 2.1 實作 Configuration file compatibility：依 design 決策「conf 相容解析器只支援生成子集，不模擬 bash」實作 internal/conf：內建預設值表、`VAR=value`/`VAR="value"`/`VAR='value'` 賦值、註解與空行略過、不支援行靜默略過、`. "$_VL_DIR/themes/<name>.conf"` source 行就地展開（`$_VL_DIR` 解析為 renderer 執行檔所在目錄，對齊 bash 版 `_VL_DIR="${0%/*}"`＝statusline.sh 所在目錄；真實部署為 `~/.claude/coralline`，在 conf 目錄下一層——迴歸測試必須用這個雙層佈局，不得用 conf 與 themes 同層的攤平佈局）、`VL_CONFIG` 環境變數覆寫路徑、後者覆寫前者的優先序；驗證：`go test ./internal/conf` 覆蓋含主題 source 的使用者 conf、單雙引號裸值、未知構文略過三類案例全綠

## 3. 輸入管線（Render statusline from stdin session JSON / Bounded stdin consumption）

- [x] 3.1 實作 Render statusline from stdin session JSON 與 Bounded stdin consumption：stdin 讀取上限 4 MiB（超出截斷續行）、design 契約列出的欄位以 encoding/json 抽取、缺欄位與壞損 JSON 一律降級為空值/零值且不 panic、stdout 不出現錯誤訊息；驗證：`go test ./internal/inputjson` 覆蓋完整 JSON、缺欄位、壞損 JSON、>4MiB 輸入四類案例全綠

## 4. 程序安全（Hard process deadline (watchdog)）

- [x] 4.1 實作 Hard process deadline (watchdog)：依 design 決策「watchdog 以 goroutine 硬死線實作，逾時靜默退出」在 main 起始布署 5 秒 time.AfterFunc → os.Exit(1)（無輸出）；驗證：以寫端保持開啟、永不 EOF 的 stdin 管線啟動 coralline.exe，程序於 5 秒內自行退出、exit code 1、stdout 為空（手動或 script 斷言）
- [x] 4.2 依 design 決策「git 子程序帶 context 逾時與 GIT_OPTIONAL_LOCKS=0」實作 internal/gitstate：exec.CommandContext 以 2.5 秒逾時執行 git -C <cwd> status --porcelain=v2 --branch、env 帶 GIT_OPTIONAL_LOCKS=0、逾時 Kill 子程序不留孤兒、任何失敗時 git segment 資料為空（Git child process is time-bounded and lock-free）；解析 porcelain v2 的 branch.oid/branch.head/branch.ab 與 staged/unstaged/untracked 標記；驗證：`go test ./internal/gitstate` 以假輸出覆蓋解析案例，逾時行為以可注入的慢命令測試

## 5. 渲染核心（Core segment set / Fixed multi-line layout）

- [x] 5.1 實作 internal/render 基元：ANSI 256 / truecolor fg/bg、bar 填充公式 (pct*width+50)/100 取整封頂、token 縮寫（999→999、1234→1.2k、1234567→1.2M）、VL_WARN_PCT/VL_HOT_PCT 門檻換色、pill 膠囊組裝（cap/分隔）；驗證：`go test ./internal/render` 對 bar 四捨五入邊界與 token 縮寫表逐案斷言
- [x] 5.2 實作 Core segment set 與 Fixed multi-line layout：八個核心 segment（dir/git/model/effort/ctx/limit5h/limit7d/burn）之內容與隱藏規則（空資料隱藏、未實作 segment 名靜默略過、VL_CTX_TOKENS=0 時 ctx 不含 token 明細），以及 VL_SEGMENTS/VL_SEGMENTS2/VL_SEGMENTS3 + VL_MAX_LINES 的固定多行版面（整行皆隱藏則整行省略）；驗證：`go test ./internal/render` 的 golden test 以 testdata/ 固定輸入比對整行輸出

## 6. 資料檔相容（依 design 決策「資料檔演算法自 bash 參考實作逐一移植，格式逐位元組相容」）

- [x] 6.1 實作 usage-state.json byte-compatible export：VL_USAGE_STATE=1 且 five_hour pct 非空時，以固定欄位順序單行 JSON + 換行、temp+rename 原子寫入、孤兒 tmp 清掃、VL_USAGE_STATE_FILE 路徑覆寫；驗證：`go test ./internal/usage` 斷言序列化逐位元組等於 design 契約範例字串
- [x] 6.2 實作 Rate-limit snapshot directory compatibility：limit-5h.d / limit-7d.d 空目錄快照（名稱 %010d_%07.3f）、讀取取最大 reset、GC 其餘、修剪 now+視窗上限外的中毒哨兵；驗證：`go test ./internal/usage` 覆蓋命名編碼（1770000000/41.25→1770000000_041.250）、GC 與哨兵修剪案例
- [x] 6.3 實作 burn-5h.tsv format and algorithm compatibility：三欄 TSV 追加、awk 參考演算法移植（僅當前視窗樣本、整數百分比跨越點、≥2 跨越且時距≥視窗十分之一才給斜率、warming/idle/active 三態、哨兵列丟棄自癒、實體列數修剪 + 原子改寫）；驗證：`go test ./internal/usage` 對三態各有案例、修剪與哨兵案例全綠

## 7. 整合驗證與上線

- [x] 7.1 依 design 決策「以 bash 版為 oracle 的視覺等價驗證」：以 ~/.claude/coralline/sample-input.json 同時餵 bash 版與 Go 版，比對輸出視覺等價（容許差異僅限時變欄位），並將該輸入納入 testdata/ golden test；驗證：比對結果零非預期差異 + `go test ./...` 全綠
- [x] 7.2 殭屍與程序面驗收（Zero MSYS-family child processes）：render 期間以 Get-CimInstance 觀測程序樹僅含 coralline.exe 與至多一個 git.exe、無 bash/awk/coreutils；連同 4.1 的永不 EOF 情境一併留下驗證紀錄；驗證：觀測輸出貼入 change 目錄備查
- [x] 7.3 部署與註冊：建置 coralline.exe 複製至 ~/.claude/coralline/，於 Claude Code 使用者 settings.json 寫入 statusLine 註冊指向該 exe（回滾 = 改回 bash 命令）；驗證：Claude Code 實際顯示三行 pill 狀態列，且 usage-state.json 持續更新（Stop-hook usage guard 不需任何修改）
