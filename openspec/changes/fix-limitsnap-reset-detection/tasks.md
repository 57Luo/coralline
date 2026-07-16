## 1. 紅燈測試（先寫、先確認失敗）

- [x] 1.1 依 spec Requirement「Same-window usage drop triggers high-water reset」在 internal/usage/limitsnap_test.go 新增測試：既有 `1784311200_075.000` 項目、Sample 寫入同 epoch 且 pct=3.0 → Latest 回傳 003.000；以及抖動情境：同 epoch pct=74.5 → 高水位維持 075.000。執行 `go test ./internal/usage/ -run TestSample` 確認驟降案例因舊行為（75 獲勝）而失敗、失敗原因正確
- [x] 1.2 依 spec Requirement「Reset detection is scoped to a single reset epoch」新增測試：既有 `1784311200_075.000`、Sample 寫入不同 epoch 1784916000 且 pct=2.0 → 不觸發清除，兩個 epoch 的項目並存（讀取時仍由較大 epoch 獲勝的既有邏輯驗證不變）。執行 `go test ./internal/usage/` 確認此測試在現行程式下的通過/失敗狀態符合預期（此為行為不變的守護測試，允許立即通過）

## 2. 實作重置偵測

- [x] 2.1 依 design 決策「在 Sample 寫入端偵測重置，而非 Latest 讀取端」與「重置判定：同 reset epoch 且百分比下降超過 5 個百分點」修改 internal/usage/limitsnap.go 的 Sample：寫入前列出 store 中同 reset epoch 的既有項目，若既有最高 pct − 新 pct > 5.0 則刪除該 epoch 全部既有項目再寫入；行為成立的判準為 1.1 的驟降測試轉綠、抖動測試維持綠
- [x] 2.2 依 spec Requirement「Reset purge failures stay silent」確保清除過程的 os.Remove 錯誤被靜默忽略且新樣本照常寫入（沿用檔內既有的錯誤忽略慣例），並新增併發刪除情境測試：purge 目標已不存在時 Sample 仍成功寫入。以 `go test ./internal/usage/` 驗證

## 3. 回歸驗證

- [x] 3.1 執行 `go test ./...` 全數通過，確認 Latest、burn（BurnEstimate 讀取同一 7d store）與 golden render 測試無回歸；design 決策「否決替代方案：reset tombstone 標記」不需程式碼，僅確認未誤引入任何 tombstone/標記檔案寫入
