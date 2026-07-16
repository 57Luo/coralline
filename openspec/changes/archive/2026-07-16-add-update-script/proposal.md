## Why

上游脫鉤後本 repo 是唯一發行來源，且 repo checkout 本身就是發行媒介——但目前「更新已安裝的 statusline」是口耳相傳的手動三步驟（pull、rebuild、放置），沒有測試閘門，步驟會漏、build 錯路徑不會叫。可重複的操作應該是 repo 內的腳本，不是文件裡的教學。

## What Changes

- 新增 update 腳本（Windows 用 PowerShell、macOS/Linux 用 sh，各一份薄殼），冪等地執行：git pull → go test ./... → go build 到 statusline 安裝路徑（`~/.claude/coralline/coralline[.exe]`）→ repo themes 有變更時同步複製到安裝目錄。任一步失敗立即中止並以非零 exit code 結束，不留下半更新狀態（build 失敗時不覆蓋既有 binary）。
- install.sh 與 configure.sh 凍結不動：不刪除、不加 Go 功能（服務未來公開發行情境，非本 change 範圍）。

## Non-Goals

- 不改 install.sh／configure.sh（凍結）。
- 不做版本檢查、自動排程、或跨機器同步——腳本由使用者手動執行。
- 不處理首次安裝（設定檔產生仍走 configure.sh）；腳本假設 `~/.claude/coralline/` 已存在。

## Capabilities

### New Capabilities

- `self-update`: repo 內建的 statusline 更新腳本——單一指令將已安裝的 binary 與 themes 同步到 repo 最新狀態，測試不過不部署

### Modified Capabilities

(none)

## Impact

- Affected specs: 新增 `self-update`
- Affected code:
  - New: update.ps1, update.sh
  - Modified: (none)
  - Removed: (none)

