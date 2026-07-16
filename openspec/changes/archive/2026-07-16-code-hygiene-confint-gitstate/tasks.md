## 1. confInt 合併

- [x] 1.1 在 internal/conf 新增 int 解析方法（語意與現行 confInt 完全相同：值為空或解析失敗時回傳預設值），並在 internal/conf/conf_test.go 新增測試涵蓋三種情況（有效整數、空值回退、非數字回退）。先寫測試確認紅燈（方法尚不存在，編譯失敗即為紅燈），再實作轉綠
- [x] 1.2 cmd/coralline/main.go 與 internal/render/segments.go 改用 conf 的新方法，刪除兩份重複的 confInt。驗證：`grep -rn "func confInt"` 在 repo 中無結果，`go test ./...` 全數通過，golden test 不變

## 2. gitstate 註解修正

- [x] 2.1 修正 cmd/coralline/main.go 中 git 收集觸發處的註解：從「單一 git 子行程」改為如實描述 gitstate.Run 啟動三個子行程（status、rev-parse --show-toplevel、rev-list --count）並共享單一逾時 context。驗證：內容審閱註解與 internal/gitstate/gitstate.go 的實際行為一致，`go build ./...` 通過

