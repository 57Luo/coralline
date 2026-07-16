## 1. Input 解析擴展

- [x] 1.1 inputjson.Input struct 新增 Cost (string), LinesAdd (int64), LinesDel (int64), OutStyle (string), DurMs (int64) 欄位，raw struct 新增對應 JSON path 映射（cost.total_cost_usd, cost.total_lines_added, cost.total_lines_removed, output_style.name, cost.total_duration_ms）。Parse() 填充新欄位。驗證：inputjson_test.go 新增 test case 確認新欄位正確解析，go test ./internal/inputjson/... 通過。

## 2. 簡單 JSON 格式化 Segments

- [x] 2.1 實作 cost segment：讀取 Input.Cost，按 VL_COST_DECIMALS 格式化為 USD 字串，空值或 "0" 時靜默。驗證：segments_test.go 新增 cost segment test case，go test ./internal/render/... 通過。
- [x] 2.2 實作 clock segment：讀取 VL_CLOCK 設定（24h/12h/off）和 VL_CLOCK_SECONDS，格式化當前時間。驗證：segments_test.go 新增 clock segment test case，go test ./internal/render/... 通過。
- [x] 2.3 實作 lines segment：讀取 Input.LinesAdd/LinesDel，零值時靜默，使用 FG_OK/FG_HOT 上色。驗證：segments_test.go 新增 lines segment test case，go test ./internal/render/... 通過。
- [x] 2.4 實作 tokens segment：讀取 Input 的 token 欄位，用 fmt_tok 邏輯格式化，DIM 前景色。驗證：segments_test.go 新增 tokens segment test case，go test ./internal/render/... 通過。
- [x] 2.5 實作 style segment：讀取 Input.OutStyle，"default" 或空值時靜默。驗證：segments_test.go 新增 style segment test case，go test ./internal/render/... 通過。
- [x] 2.6 實作 duration segment：讀取 Input.DurMs，格式化為人類可讀時間字串（與 bash fmt_duration 一致），零值時靜默。驗證：segments_test.go 新增 duration segment test case，go test ./internal/render/... 通過。

## 3. Git 交互 Segments

- [x] 3.1 gitstate.State 新增 StashCount int 欄位，gitstate.Run() 在 git repo 內執行 git rev-list --walk-reflogs --count refs/stash 填充。失敗時 StashCount 為 0。驗證：gitstate_test.go 新增 stash count test case，go test ./internal/gitstate/... 通過。
- [x] 3.2 實作 stash segment：讀取 gitstate.State.StashCount，零值或非 git repo 時靜默。驗證：segments_test.go 新增 stash segment test case，go test ./internal/render/... 通過。
- [x] 3.3 實作 project segment：讀取 gitstate.State.Root 取 filepath.Base，非 git repo 時若 dir 不在 segment list 則 fallback 到 dir 行為，受 VL_NAME_MAX 截斷。驗證：segments_test.go 新增 project segment test case，go test ./internal/render/... 通過。

## 4. Runtime 偵測 Segments

- [x] 4.1 新增 internal/runtime/ package，實作 DetectNode(cwd string, probe bool) string：walk 祖先目錄找 .nvmrc/.node-version，strip v prefix；probe=true 時 fallback 到 node --version。驗證：runtime_test.go 用 temp dir 和 pin file 測試 walk 邏輯，go test ./internal/runtime/... 通過。
- [x] 4.2 實作 DetectPython(cwd string, probe bool) string：優先 VIRTUAL_ENV basename，其次 CONDA_DEFAULT_ENV（skip "base"），其次 .python-version walk；probe=true 時 fallback 到 python3 --version。驗證：runtime_test.go 用 env var 和 temp dir pin file 測試，go test ./internal/runtime/... 通過。
- [x] 4.3 實作 node segment：呼叫 runtime.DetectNode，空字串時靜默，使用 VL_NODE_GLYPH。驗證：segments_test.go 新增 node segment test case，go test ./internal/render/... 通過。
- [x] 4.4 實作 python segment：呼叫 runtime.DetectPython，空字串時靜默，使用 VL_PY_GLYPH。驗證：segments_test.go 新增 python segment test case，go test ./internal/render/... 通過。

## 5. main.go 整合與 Segment 註冊

- [x] 5.1 cmd/coralline/main.go 的 renderStatusline 函式整合新 segment：將 runtime 偵測結果和 stash count 傳入 render pipeline，segment scan 邏輯覆蓋所有 18 個 segment 名稱。驗證：go build ./cmd/coralline 成功，以 test/sample-input.json（擴展新欄位後）為 input 執行 Go binary 輸出包含新 segment。

## 6. Golden Test 與 Bash 對照

- [x] 6.1 擴展 test/sample-input.json 加入 cost、lines、output_style、duration 欄位。更新 golden_test.go 的 test case 涵蓋新 segment。驗證：go test ./internal/render/... -run Golden 通過，且 golden output 與 bash statusline.sh 對相同 input 的輸出一致。

## 7. configure.sh Hook 切換

- [x] 7.1 configure.sh 的 update_settings() 新增 Go binary 偵測：檢查 $TARGET_DIR/coralline 或 coralline.exe 是否存在且可執行，存在時 hook command 指向 Go binary，否則 fallback 到 bash statusline.sh。驗證：手動測試 — 放置假的可執行 binary 後執行 configure.sh --install-only，確認 settings.json 的 statusLine.command 指向 Go binary；移除後重跑確認 fallback 到 bash。
