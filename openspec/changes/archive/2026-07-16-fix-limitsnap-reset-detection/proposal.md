## Problem

開啟 limit-sync 時，statusline 的 5h/7d 用量採「跨 session 高水位」策略：每次渲染把讀數存入 `limit-*.d/` 目錄集，顯示時取最大值。2026-07-16 Anthropic 對 rate limit 進行臨時重置後，7d 用量從 75% 掉到 3%，但 `resets_at` 維持同一個值；舊的 75% 快照因為排序永遠獲勝，statusline 卡在 75% 且不再更新，直到窗口自然結束（可能長達數天）。

## Root Cause

快照名稱為 `<reset_epoch>_<pct>`，`Latest` 取字典序最大者並刪除其餘項目。此設計隱含不變量「同一 `resets_at`（同一窗口）內百分比單調遞增」，因此取最大值安全。官方臨時重置打破了這個不變量：`resets_at` 不變、百分比驟降。之後每次寫入的新樣本（如 `..._003.000`）都排在舊高水位（`..._075.000`）之前，被 `Latest` 當成非最大值刪除，舊值永久獲勝。

## Proposed Solution

在 `Sample`（寫入端）加入重置偵測：寫入新樣本前，若 store 中存在**相同 reset epoch** 的既有項目，且既有最高百分比比新樣本高出超過閾值（5 個百分點），判定為官方重置——先刪除該 reset epoch 的所有既有項目，再寫入新樣本。閾值用來吸收跨 session 的讀數抖動（微小落差不觸發清除）。

已知的殘餘現象：閒置 session 可能在重置後仍回報舊百分比，重新污染 store，造成短暫的新舊值交替，直到該 session 取得新資料為止；下一個新樣本會再次觸發清除。此為可接受的暫態（詳見 design 的替代方案討論）。

## Success Criteria

- 同一 reset epoch 下，新樣本比既有高水位低超過 5 個百分點時：既有項目被清除，`Latest` 回傳新樣本的百分比。
- 同一 reset epoch 下，新樣本僅比既有高水位低 5 個百分點以內（抖動）：既有高水位保留，行為與現行相同。
- 不同 reset epoch 的既有項目不受重置偵測影響（新窗口取代舊窗口的既有邏輯不變）。
- `go test ./internal/usage/` 全數通過，含新增的重置偵測測試。

## Capabilities

### New Capabilities

- `limit-snapshot-reset-detection`: limit-sync 高水位快照在同一 reset 窗口內偵測官方用量重置並清除過期高水位

### Modified Capabilities

(none)

## Impact

- Affected specs: 新增 `limit-snapshot-reset-detection`
- Affected code:
  - Modified: internal/usage/limitsnap.go, internal/usage/limitsnap_test.go
  - New: (none)
  - Removed: (none)
