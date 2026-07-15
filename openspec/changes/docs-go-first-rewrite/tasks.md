## 1. INSTALL.md 重寫

- [x] 1.1 重寫 INSTALL.md：主要路徑改為 Go renderer — 文件開頭即為「go build 編譯 coralline.exe + 在 Claude Code settings.json 的 statusLine.command 註冊該 .exe」的完整步驟（含 Windows 路徑範例與驗證方式：執行 .exe 並以 example/ 下的範例 stdin JSON 確認有輸出）。實現 Requirement: Documentation presents the Go renderer as the primary path。驗證：照文件從零走完 Go 路徑可得到可運作的 statusline 註冊，且文件中 Go 路徑出現在任何 bash 流程之前。
- [x] 1.2 將既有 install.sh bash 安裝流程完整移入 INSTALL.md 的附錄章節（標題明示為 bash 版/相容路徑），並敘明兩版共用同一份 coralline.conf 與 themes。實現 Requirement: Bash version is retained as an appendix。驗證：附錄含原 install.sh 三種安裝方式的完整說明，內容足以不翻 git 歷史即完成安裝。

## 2. README 重寫

- [x] 2.1 重寫 README.md：開頭段落先陳述本 fork 定位（熱路徑改寫為 Go 原生 .exe）與動機（Windows MSYS 殭屍程序事故，一句話並連結 openspec/changes/go-renderer-core/proposal.md），再進入功能介紹；安裝章節指向 INSTALL.md 的 Go 主路徑。實現 Requirement: Documentation presents the Go renderer as the primary path。驗證：README.md 開頭第一個章節即含 fork 定位與 proposal 連結，且 bash 指令未出現在 Go 說明之前。
- [x] 2.2 README.md 的 segment 總表保留全部 16 個 segment 並新增 Go 支援狀態欄：ctx、git、dir、model、effort、limit5h、limit7d、burn 標為支援，其餘標為僅 bash 版；表格後說明 lean/classic 風格目前僅 bash 版可用、未移植功能一律指向 bash 附錄。實現 Requirement: Go coverage is stated honestly。驗證：表格恰好 16 列、支援標記與 openspec/changes/go-renderer-core/proposal.md 所列 8 個 segment 一致。
- [x] 2.3 重寫 README.zh-TW.md 使其與重寫後的 README.md 結構對齊（每個章節在兩語言中互有對應，資訊等價而非逐句直譯）。實現 Requirement: The two README languages stay aligned。驗證：兩檔章節標題一一對應，segment 表同為 16 列且 Go 支援標記一致。

## 3. 驗證與收尾

- [x] 3.1 對三份文件做交叉一致性檢查：Go 涵蓋範圍（8 segment、pill、固定多行）在三份文件中陳述一致（Requirement: Go coverage is stated honestly）；未移植功能一律指向 bash 附錄（Requirement: Bash version is retained as an appendix）；無殘留「僅 bash 敘事」段落宣稱 Go 版不存在。驗證：逐份 grep 8 個 segment 名與 lean/classic 字樣，人工確認每處陳述與 spec 的誠實標注要求相符。
