## 1. Segment 表格修正

- [x] 1.1 依 spec Requirement「Segment support tables state actual renderer coverage」更新 README.md、README.zh-TW.md、INSTALL.md 的 segment 相容性表格：18 個 segment 全標為 Go 支援，bash-only 僅剩 lean/classic 風格與 auto layout。驗證：對照 internal/render/segments.go 的 builders 註冊表逐項核對，三份文件 grep 不到任何 segment 被標為 bash-only

## 2. URL 與升級路徑

- [x] 2.1 依 spec Requirement「Install and upgrade commands point at this repository」把 INSTALL.md、UPGRADE.md 所有安裝／升級指令的 URL 改為 57Luo/coralline。驗證：兩份文件 grep 不到 Nanako0129
- [x] 2.2 依 spec Requirement「Upgrade documentation covers Go binary users」在 UPGRADE.md 新增 Go binary 使用者的升級章節：以 update 腳本（update.ps1／update.sh，由 add-update-script change 提供）為主要升級方式，並說明既有 coralline.conf 與 themes 設定不需變更即可沿用；若實作本任務時 add-update-script 尚未完成，改寫為手動三步驟（pull、go test、go build 到安裝路徑）並註明將由 update 腳本取代。驗證：內容審閱該章節涵蓋升級方式與設定相容性兩點

## 3. Canonical spec Purpose

- [x] 3.1 依 spec Requirement「Canonical specs carry a filled Purpose section」補寫 openspec/specs/go-renderer-segments/spec.md 與 openspec/specs/limit-snapshot-reset-detection/spec.md 的 Purpose 段落（各 1-3 句）。驗證：兩檔 grep 不到 TBD

