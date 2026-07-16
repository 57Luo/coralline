## Summary

把文件與 canonical specs 拉回與 Go-first 實作一致：修正 README／INSTALL 過期的 segment 支援表、fork 安裝 URL、UPGRADE.md 的 Go 升級路徑，並補上兩個 canonical spec 的 TBD Purpose。

## Motivation

2026-07-16 的 go-segment-parity change 讓 Go renderer 補齊全部 18 個 segment，但 README.md、README.zh-TW.md、INSTALL.md 的相容性表格仍宣稱 Go 版只支援 8 個、其餘 9 個為 bash-only。此錯誤已實際誤導讀者（含 AI 代理）作出「需退回 bash renderer」的錯誤結論。另外：INSTALL.md／UPGRADE.md 的 curl 指令指向 upstream（Nanako0129/coralline）而非本 fork（57Luo/coralline），照文件安裝會取得沒有 Go renderer 的舊版；UPGRADE.md 缺 Go binary 使用者的升級指引（其重寫先前被 docs-go-first-rewrite 的 Non-Goals 延後，延後條件「Go 版功能補齊」現已成立）；openspec/specs/go-renderer-segments/spec.md 與 openspec/specs/limit-snapshot-reset-detection/spec.md 的 Purpose 仍是 TBD 佔位字。

## Proposed Solution

- 三份文件的 segment 表格改為如實陳述：全部 18 個 segment Go 版皆支援，bash-only 僅剩 lean/classic 風格與 auto layout。
- INSTALL.md／UPGRADE.md 的安裝／升級 URL 一律改指本 repo（57Luo/coralline）——上游（Nanako0129/coralline）已停止追蹤，本 repo 即為事實上的主線。
- UPGRADE.md 增加 Go binary 使用者的升級路徑（何時需要重新 go build、設定檔相容性說明）。
- 補寫兩個 canonical spec 的 Purpose 段落（各 1-3 句，描述該 capability 的範圍）。

## Non-Goals

- 不改任何程式碼、install.sh、configure.sh 的行為（install.sh 的 Go binary 策略是另一個待討論的產品決策）。
- 不重寫文件整體結構，只修正錯誤與缺漏。
- 不處理 assets/ 截圖是否過期。

## Alternatives Considered

- 「等 install.sh 支援 Go binary 後一起改文件」：否決——文件現在就在誤導使用者，等待另一個未排程的決策只會延長錯誤存活時間。

## Capabilities

### New Capabilities

- `docs-accuracy`: 使用者文件（README、INSTALL、UPGRADE）與 canonical specs 對實作能力的陳述必須與程式碼現狀一致

### Modified Capabilities

(none)

## Impact

- Affected specs: 新增 `docs-accuracy`；補寫 openspec/specs/go-renderer-segments/spec.md 與 openspec/specs/limit-snapshot-reset-detection/spec.md 的 Purpose
- Affected code:
  - Modified: README.md, README.zh-TW.md, INSTALL.md, UPGRADE.md, openspec/specs/go-renderer-segments/spec.md, openspec/specs/limit-snapshot-reset-detection/spec.md
  - New: (none)
  - Removed: (none)

