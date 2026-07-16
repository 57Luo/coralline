## 1. 紅燈測試

- [x] 1.1 依 spec Requirement「Path collapsing supports Windows separators」在 internal/render/segments_test.go 新增 collapsePath 參數化測試，逐列採用 spec Example 表格的四組值（兩組 Windows 收合、一組 POSIX 不收合、一組 POSIX 收合）。執行 `go test ./internal/render/ -run TestCollapsePath` 確認 Windows 兩列失敗（現行以 `/` 切割而 no-op）、POSIX 兩列通過

## 2. 實作

- [x] 2.1 修改 internal/render/segments.go 的 collapsePath：輸入含 `\` 時以 `\` 切割與重組，否則維持 `/`；POSIX 路徑輸出 byte-for-byte 不變。驗證：1.1 全部轉綠，且 `go test ./internal/render/` 全數通過（golden test 不需修改）

## 3. 回歸驗證

- [x] 3.1 執行 `go test ./...` 全數通過；以本機真實輸入手動驗證：向 coralline.exe 餵入 workspace.current_dir 為深層 Windows 路徑（超過 VL_PATH_DEPTH）的 session JSON，確認 dir segment 顯示收合後的路徑

