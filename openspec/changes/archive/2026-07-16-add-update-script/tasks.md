## 1. 腳本實作

- [x] 1.1 依 spec Requirement「Single-command statusline update」新增 update.ps1：git pull 當前分支 → go test ./... → go build 到暫存檔、成功後才覆蓋 `$env:USERPROFILE\.claude\coralline\coralline.exe`（build 到暫存再搬移，保證失敗不動舊 binary）→ 比對 themes/ 與安裝目錄 themes 差異、有差異才複製；任一步失敗即以非零 exit code 中止。驗證：在本機實際執行一次成功更新；再以人為弄壞一個測試驗證「Test failure blocks deployment」情境（binary 時間戳不變）後還原
- [x] 1.2 依 spec Requirement「Single-command statusline update」新增 update.sh（POSIX 版，行為與 1.1 相同，安裝路徑 `~/.claude/coralline/coralline`）。驗證：bash -n 語法檢查通過；邏輯與 update.ps1 逐步對照一致（本機無 POSIX 環境，實機驗證留待另一台機器首次使用時）
- [x] 1.3 依 spec Requirement「Update script is idempotent」驗證連續執行兩次 update.ps1：第二次在無新 commit 下正常完成、exit code 0、statusline 渲染正常

## 2. 文件連動

- [x] 2.1 README.md 與 README.zh-TW.md 的更新說明段落加入 update 腳本用法（一行指令）。驗證：內容審閱兩份 README 都提及 update.ps1／update.sh 及其行為（測試不過不部署）

