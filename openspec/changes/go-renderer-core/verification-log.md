# go-renderer-core 驗證紀錄

日期：2026-07-14 ／ 機器：Windows 11 Pro（AMD64）／ go1.26.5

## Task 4.1 — watchdog（Hard process deadline）

指令：`bash scripts/watchdog_test.sh ./coralline.exe`（stdin 以 `< <(sleep 20)` 保持開啟、永不 EOF）

```
PASS: watchdog exited in 5s with code 1 and empty stdout
```

## Task 7.1 — bash oracle 視覺等價

隔離 harness（scratchpad，雙層真實部署佈局：conf 於上層、statusline.sh / coralline.exe / themes 於 coralline/ 子目錄），輸入為 sample-input.json，兩版各自執行後 `diff`：

```
IDENTICAL (byte-for-byte)
```

（比 spec 要求的「視覺等價」更強；三行 pill、catppuccin-mocha 主題色、ctx/git｜dir/model/effort｜limit5h/limit7d/burn 版面一致。）

usage-state.json 交叉比對：兩版輸出僅 `updated_at` 相差 1 秒（執行時間差，時變欄位），其餘逐位元組一致。

注意事項：bash 版以相對路徑呼叫（`bash coralline/statusline.sh`）時 `_VL_CONFIG_DIR` 推導會失敗而 fallback 預設值——bash 版既有行為，非 Go 版缺陷；oracle 比對以絕對路徑呼叫。

## Task 7.2 — Zero MSYS-family child processes

方法：PowerShell 啟動 30 次 render（cwd＝本 repo，git segment 啟用），每 5ms 輪詢 `Get-Process`，比對名單 bash/sh/awk/gawk/jq/grep/sort/ls/date/stty/mintty/git-bash/coreutils：

```
renders: 30
MSYS-family processes observed: NONE
git.exe observed: True
exit code last render: 0
stdout bytes: 764
```

結論：render 期間唯一子程序為原生 git.exe，零 MSYS 家族程序。

## 上線後缺陷 #2 — 實況 stdin 型別差異（已修）

症狀：實際掛上 Claude Code 後只顯示 limit5h/limit7d 行。診斷：以 tee 包裝取得實況 payload，發現 `rate_limits.*.resets_at` 是**數字 epoch**（sample-input.json 是 ISO 字串），Go struct 宣告為 string → 整包 Unmarshal 失敗 → 全欄位降級；limit 段靠磁碟快照 fallback 存活，形成誤導性症狀。修復：`internal/inputjson` 新增 `flexStr`（字串/數字通吃、逐欄位寬容），spec 補「type variation MUST NOT degrade any other field」需求與實況 Example；迴歸測試 TestNumericResetsAt / TestStringTypedNumbersAccepted。以凍結的實況 payload 重放驗證三行全渲染、5h 92% 紅色正確。教訓與缺陷 #1 相同：sample/測試資料不夠代表真實生產輸入。

## 測試套件

`go build ./...` 綠；`go test ./... -count=1` 全綠、零 skip（含 verifier 抓出 `$_VL_DIR` 語意缺陷後補上的雙層佈局迴歸測試）。
