# go-renderer-data-compat Specification

## Purpose

Keep the Go renderer's on-disk data formats bit-compatible with the bash implementation — the usage-state JSON snapshot, burn sample TSV, and the mkdir-based rate-limit snapshot directory stores — so both renderers (and external hooks) can read and write the same files on the same machine without migration.

## Requirements

### Requirement: usage-state.json byte-compatible export

When VL_USAGE_STATE=1 and the five-hour percentage field is non-empty, the renderer SHALL write a single-line JSON file (default `<config-dir>/usage-state.json`, path overridable via VL_USAGE_STATE_FILE) with exactly this field order and shape: `{"source":"coralline","updated_at":<epoch>,"model":"<model>","five_hour":{"used_percentage":<pct>,"resets_at":"<str>"},"seven_day":{"used_percentage":<pct>,"resets_at":"<str>"}}` followed by a newline. The write MUST be atomic (temp file then rename), and orphaned temp files from dead sessions SHALL be swept. External consumers of this file (the Stop-hook usage guard) MUST NOT require any change.

#### Scenario: Snapshot written atomically on render

- **WHEN** a render runs with VL_USAGE_STATE=1 and rate_limits.five_hour.used_percentage present
- **THEN** usage-state.json contains the single-line JSON with the exact field order above and no temp file remains

##### Example: serialized output

- **GIVEN** epoch 1770000000, model "Fable 5", five_hour 41/"2026-07-14T10:00:00Z", seven_day 79/"2026-07-18T00:00:00Z"
- **WHEN** the snapshot is written
- **THEN** the file contains `{"source":"coralline","updated_at":1770000000,"model":"Fable 5","five_hour":{"used_percentage":41,"resets_at":"2026-07-14T10:00:00Z"},"seven_day":{"used_percentage":79,"resets_at":"2026-07-18T00:00:00Z"}}`


<!-- @trace
source: go-renderer-core
updated: 2026-07-16
code:
  - scripts/watchdog_test.sh
  - internal/inputjson/inputjson.go
  - internal/render/testdata/input.json
  - configure.sh
  - go.mod
  - internal/runtime/runtime.go
  - internal/usage/usagestate.go
  - internal/render/testdata/golden_pill.txt
  - INSTALL.md
  - README.zh-TW.md
  - internal/render/layout.go
  - internal/render/testdata/coralline.conf
  - internal/conf/conf.go
  - internal/usage/limitsnap.go
  - internal/render/render.go
  - internal/usage/usage.go
  - internal/render/testdata/themes/catppuccin-mocha.conf
  - internal/conf/defaults.go
  - internal/epoch/epoch.go
  - cmd/coralline/main.go
  - internal/render/segments.go
  - internal/usage/burn.go
  - README.md
  - internal/gitstate/gitstate.go
tests:
  - internal/render/render_test.go
  - internal/usage/limitsnap_test.go
  - internal/render/segments_test.go
  - internal/usage/burn_test.go
  - internal/gitstate/gitstate_test.go
  - internal/usage/usagestate_test.go
  - internal/conf/conf_test.go
  - internal/render/golden_test.go
  - internal/inputjson/inputjson_test.go
  - internal/runtime/runtime_test.go
-->

---
### Requirement: Rate-limit snapshot directory compatibility

When VL_LIMIT_SYNC=1 the renderer SHALL share the bash implementation's snapshot stores `limit-5h.d` and `limit-7d.d`: each snapshot is an empty directory named `<reset-epoch as %010d>_<pct as %07.3f>`. On write the renderer SHALL create such a directory; on read it SHALL select the entry with the greatest reset epoch, remove all other entries, and prune poisoned entries whose reset epoch exceeds now plus the window ceiling. Reads and writes MUST be safe against a concurrent bash-version session operating on the same store.

#### Scenario: Cross-implementation round trip

- **WHEN** the Go renderer writes a snapshot and the bash implementation subsequently reads the store (or vice versa)
- **THEN** the reader obtains the same percentage and reset epoch that were written, and the store ends with a single surviving entry

##### Example: directory name encoding

- **GIVEN** reset epoch 1770000000 and percentage 41.25
- **WHEN** a snapshot is written
- **THEN** the created directory is named `1770000000_041.250`


<!-- @trace
source: go-renderer-core
updated: 2026-07-16
code:
  - scripts/watchdog_test.sh
  - internal/inputjson/inputjson.go
  - internal/render/testdata/input.json
  - configure.sh
  - go.mod
  - internal/runtime/runtime.go
  - internal/usage/usagestate.go
  - internal/render/testdata/golden_pill.txt
  - INSTALL.md
  - README.zh-TW.md
  - internal/render/layout.go
  - internal/render/testdata/coralline.conf
  - internal/conf/conf.go
  - internal/usage/limitsnap.go
  - internal/render/render.go
  - internal/usage/usage.go
  - internal/render/testdata/themes/catppuccin-mocha.conf
  - internal/conf/defaults.go
  - internal/epoch/epoch.go
  - cmd/coralline/main.go
  - internal/render/segments.go
  - internal/usage/burn.go
  - README.md
  - internal/gitstate/gitstate.go
tests:
  - internal/render/render_test.go
  - internal/usage/limitsnap_test.go
  - internal/render/segments_test.go
  - internal/usage/burn_test.go
  - internal/gitstate/gitstate_test.go
  - internal/usage/usagestate_test.go
  - internal/conf/conf_test.go
  - internal/render/golden_test.go
  - internal/inputjson/inputjson_test.go
  - internal/runtime/runtime_test.go
-->

---
### Requirement: burn-5h.tsv format and algorithm compatibility

The renderer SHALL append and trim the burn sample file using the bash implementation's format: one row per sample, three tab-separated columns `<epoch>\t<pct>\t<reset-epoch>`. The ETA computation SHALL be a faithful port of the awk reference: samples restricted to the current window (greatest reset epoch), integer-percent crossings counted within the sliding window, a slope accepted only when at least two crossings span at least one tenth of the window, producing state `active` with an ETA, else `idle` when crossings exist only outside the window, else `warming`. Rows whose reset epoch exceeds now plus the window ceiling SHALL be dropped (self-healing), and the file SHALL be trimmed by physical row count with an atomic rewrite. A file written by either implementation MUST remain readable and semantically intact for the other.

#### Scenario: States derived from crossing analysis

- **WHEN** the sample file for the current window contains at least two integer-percent up-crossings spanning at least one tenth of the burn window
- **THEN** the burn segment reports state active with a finite ETA computed from the crossing slope

#### Scenario: Alternating implementations do not corrupt the file

- **WHEN** renders alternate between the bash implementation and the Go renderer against the same burn-5h.tsv
- **THEN** every row remains a valid three-column TSV record and neither implementation loses the other's samples beyond the trim policy

<!-- @trace
source: go-renderer-core
updated: 2026-07-16
code:
  - scripts/watchdog_test.sh
  - internal/inputjson/inputjson.go
  - internal/render/testdata/input.json
  - configure.sh
  - go.mod
  - internal/runtime/runtime.go
  - internal/usage/usagestate.go
  - internal/render/testdata/golden_pill.txt
  - INSTALL.md
  - README.zh-TW.md
  - internal/render/layout.go
  - internal/render/testdata/coralline.conf
  - internal/conf/conf.go
  - internal/usage/limitsnap.go
  - internal/render/render.go
  - internal/usage/usage.go
  - internal/render/testdata/themes/catppuccin-mocha.conf
  - internal/conf/defaults.go
  - internal/epoch/epoch.go
  - cmd/coralline/main.go
  - internal/render/segments.go
  - internal/usage/burn.go
  - README.md
  - internal/gitstate/gitstate.go
tests:
  - internal/render/render_test.go
  - internal/usage/limitsnap_test.go
  - internal/render/segments_test.go
  - internal/usage/burn_test.go
  - internal/gitstate/gitstate_test.go
  - internal/usage/usagestate_test.go
  - internal/conf/conf_test.go
  - internal/render/golden_test.go
  - internal/inputjson/inputjson_test.go
  - internal/runtime/runtime_test.go
-->