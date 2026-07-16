## Summary

小型程式碼衛生：合併重複的 confInt 輔助函式為單一來源，並修正 main.go 中「單一 git 子行程」的過期註解。

## Motivation

`confInt`（讀取設定值並解析為 int、失敗回退預設）在 cmd/coralline/main.go 與 internal/render/segments.go 各有一份相同實作——「one fact, one source」違反，未來語意調整需改兩處且可能靜默分歧。另外 cmd/coralline/main.go 觸發 git 收集處的註解宣稱「單一 git 子行程」，實際 gitstate.Run 一旦觸發會啟動三個子行程（status、rev-parse、rev-list，共享同一個逾時 context）——註解與行為不符，誤導讀碼者評估每次渲染的成本。

## Proposed Solution

- confInt 移到 internal/conf 作為 Config 的方法（或套件函式），main.go 與 segments.go 改為呼叫同一實作，刪除兩份重複。
- main.go 的註解改為如實描述：git 收集觸發時啟動三個子行程、共享單一逾時預算。

## Non-Goals

- 不改 confInt 的解析語意（空值回退、非數字回退行為不變）。
- 不改 gitstate.Run 的行為、子行程數量或逾時設計——只修註解。
- 不處理 gitstate 的 rootCmd/stashCmd 缺測試 hook 的問題（另案）。

## Alternatives Considered

- confInt 留在 render 套件、main.go 引用：否決——conf 是兩者共同的下游依賴，語意上「解析設定值」屬於 conf 的職責。

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

(none)

## Impact

- Affected specs: 無（純內部重構與註解修正，無 spec 層行為變更）
- Affected code:
  - Modified: cmd/coralline/main.go, internal/render/segments.go, internal/conf/conf.go, internal/conf/conf_test.go
  - New: (none)
  - Removed: (none)

