## Summary

以 Go renderer 為主軸全面重寫三份使用者文件（README.md、README.zh-TW.md、INSTALL.md），bash 版降為附錄/相容說明。

## Motivation

此 fork 已因 Windows MSYS 殭屍程序事故（背景見 openspec/changes/go-renderer-core/proposal.md）將 statusline 熱路徑改寫為 Go 原生 .exe，但三份文件仍完全以 bash 版為主要敘事：安裝章節只講 install.sh 的 bash 流程，未提及 Go renderer 的存在、編譯方式、註冊方式與涵蓋範圍。讀者（包含未來的自己與 AI session）無從得知此 fork 的主要使用路徑已是 Go 版。

## Proposed Solution

- README.md 與 README.zh-TW.md 重寫：開頭即說明此 fork 的定位（Go 熱路徑改寫）、為什麼（MSYS 事故一句話帶過並連到 proposal）、Go 版目前涵蓋範圍的誠實標注 — 僅 8 個核心 segment（ctx、git、dir、model、effort、limit5h、limit7d、burn）與 pill 風格及固定多行版面；其餘 segment 與 lean/classic 風格尚未移植，需要者退回 bash 版
- segment 總表保留全部 16 個 segment，但加一欄標注 Go 版支援狀態
- INSTALL.md 重寫：主要路徑為「go build 編譯 + statusLine 註冊指向 .exe」；bash 版 install.sh 流程移至附錄章節保留
- 兩份 README 內容保持中英對齊（結構與資訊等價，非逐句直譯）
- conf/themes 設定文件維持不變的部分明確說明兩版共用同一份設定

## Non-Goals

- 不改動 configure.sh、install.sh、statusline.sh 任何程式碼
- 不撰寫尚未移植功能的文件（不預先宣稱 lean/classic 或剩餘 segment 可用）
- 不處理 UPGRADE.md（升級敘事等 Go 版功能補齊後再改）
- 不建立自動化文件檢查工具

## Impact

- Affected specs: 新增 `docs-go-first`
- Affected code:
  - Modified: README.md、README.zh-TW.md、INSTALL.md
  - New: （無）
  - Removed: （無）
