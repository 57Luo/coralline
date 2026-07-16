## Problem

dir segment 的路徑收合（超過深度時顯示 `first/second/…/last`）在 Windows 原生路徑上完全失效。實測本機 Claude Code 傳入的 `workspace.current_dir` 為反斜線形式（如 `C:\MyProject\Pershing\cgmh-trasher-recognize`），深層路徑不會被縮短，statusline 的 dir segment 可能佔滿整行。`~`（HOME 前綴）替換因為是純字串前綴比對而僥倖可用。

## Root Cause

internal/render/segments.go 的 collapsePath 以 `strings.Split(short, "/")` 切割路徑成分。Windows 原生路徑以 `\` 分隔，切割結果只有單一欄位，永遠不超過深度門檻，收合邏輯靜默 no-op。此函式是從 bash 版（只跑 POSIX 環境）直譯過來的，移植時未考慮 Go 版會直接收到 Windows 原生路徑。

## Proposed Solution

collapsePath 在切割與重組時同時支援 `/` 與 `\` 分隔符：偵測輸入路徑的主要分隔符（含 `\` 即視為 Windows 路徑），以該分隔符切割並以其重組省略形式。POSIX 路徑的既有行為（含 golden test 鎖定的輸出）完全不變。

## Non-Goals

- 不處理混合分隔符路徑（如 `C:/Users\foo`）的正規化——按主要分隔符處理即可。
- 不改 trunc、`~` 替換或 dir segment 的其他行為。
- 不改 bash 版 statusline.sh（POSIX-only，無此問題）。

## Success Criteria

- Windows 路徑 `C:\Users\demo\projects\deep\nested\coralline`（深度 4）收合為 `C:\Users\…\coralline`。
- HOME 替換後的 `~\projects\deep\nested\coralline` 同樣按 `\` 收合。
- POSIX 路徑行為不變：`/Users/demo/projects/coralline` 深度 4 時維持 `/Users/…/coralline`（golden test 不變）。
- `go test ./internal/render/` 全數通過。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `go-renderer-core`: 新增需求——dir segment 的路徑收合同時支援 POSIX 與 Windows 原生路徑分隔符（既有 POSIX 行為需求不變）

## Impact

- Affected specs: `go-renderer-core`（ADDED requirement：Windows 路徑收合）
- Affected code:
  - Modified: internal/render/segments.go, internal/render/segments_test.go
  - New: (none)
  - Removed: (none)

