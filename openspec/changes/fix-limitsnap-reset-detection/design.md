## Context

limit-sync 讓多個 Claude Code session 共享同一組 5h/7d 用量顯示：每次渲染呼叫 `usage.Sample` 把 `(reset_epoch, pct)` 以空目錄形式寫入 `limit-5h.d/`、`limit-7d.d/`，顯示時 `usage.Latest` 取 reset epoch 最大者（同 epoch 比百分比），並刪除其餘項目。此高水位策略依賴不變量「同一窗口內百分比單調遞增」。2026-07-16 官方臨時重置打破此不變量（`resets_at` 不變、百分比 75%→3%），舊高水位永久獲勝，7d 顯示卡死。

burn 投影（`internal/usage/burn.go` 的 `BurnEstimate`）也讀取同一個 7d store，同樣受惠於本修正。

## Goals / Non-Goals

**Goals:**

- 官方在窗口中途重置用量時，statusline 在下一次渲染即回到正確百分比。
- 保留既有高水位語意：正常的跨 session 讀數抖動不觸發清除。

**Non-Goals:**

- 不處理閒置 session 以過期資料重新污染 store 的暫態（見 Risks）。
- 不改動 store 的目錄集格式（`<reset_epoch>_<pct>` 空目錄），維持與 bash 實作的相容性。
- 不改動 5h/7d 以外的任何 segment 或 `Latest` 的窗口切換邏輯。

## Decisions

### 在 Sample 寫入端偵測重置，而非 Latest 讀取端

寫入端擁有「新樣本 vs 既有高水位」的完整比較資訊，且清除動作與寫入同屬一次變更，語意單純。若放在 `Latest`（讀取端），它無法區分「較低的既有項目」是舊窗口殘留還是重置後的新讀數，需要額外狀態。

### 重置判定：同 reset epoch 且百分比下降超過 5 個百分點

同一 reset epoch 表示同一窗口；窗口內百分比只應遞增，下降即異常。閾值 5 個百分點用來吸收跨 session 的讀數時間差造成的微小抖動（例如 A session 較舊的 74.9 對上 B session 的 75.0），避免誤清除。判定成立時，刪除該 reset epoch 的所有既有項目後再寫入新樣本。

### 否決替代方案：reset tombstone 標記

曾考慮在偵測到重置時寫入 tombstone 項目，讓後續過期樣本（閒置 session 回報的舊百分比）被拒收，徹底消除重新污染。否決原因：tombstone 需要定義「過期樣本」的判準（新高水位 + 閾值？時間戳？），語意模糊且引入清理負擔；而重新污染是自癒的暫態——下一個新鮮樣本會再次觸發清除，閒置 session 一旦取得新資料即停止污染。複雜度不值得。

## Implementation Contract

- **行為**：`usage.Sample(file, pctStr, resetStr, now, maxAhead)` 寫入新樣本前，檢查 store 中相同 reset epoch 的既有項目；若既有最高百分比 − 新樣本百分比 > 5.0，刪除該 reset epoch 的所有既有項目，再照常寫入新樣本。落差 ≤ 5.0 或 reset epoch 不同時，行為與現行完全相同。
- **介面／資料形狀**：`Sample` 與 `Latest` 的函式簽名不變；store 目錄集格式（`<10位reset_epoch>_<07.3f百分比>` 空目錄）不變。
- **失敗模式**：清除過程的檔案系統錯誤沿用既有慣例——靜默忽略（statusline 永不因 store 問題輸出錯誤文字）；併發渲染下重複刪除同一項目是無害的（`os.Remove` 對不存在路徑的錯誤被忽略）。
- **驗收**：`go test ./internal/usage/` 全數通過，新增測試涵蓋——(1) 同 epoch 驟降超過閾值 → 舊項目被清除、`Latest` 回傳新值；(2) 同 epoch 落差在閾值內 → 高水位保留；(3) 不同 epoch → 不觸發清除、既有窗口切換邏輯不變。
- **範圍邊界**：僅改 `internal/usage/limitsnap.go` 及其測試。`Latest`、`burn.go`、`main.go`、bash 版 statusline.sh 均不修改。

## Risks / Trade-offs

- [閒置 session 重新污染] 重置後，尚未取得新資料的 session 可能把舊百分比寫回 store，畫面短暫回跳舊值 → 下一個新鮮樣本再次觸發清除，該 session 取得新資料後自然停止；屬自癒暫態，接受。
- [真實用量驟降誤判] 理論上同窗口內百分比不會下降，若 Claude Code 端資料異常抖動超過 5 個百分點，會誤清高水位 → 影響僅為高水位暫時變低，下次讀數即恢復，無持久損害。
