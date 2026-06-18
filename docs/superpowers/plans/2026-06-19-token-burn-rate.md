# Token Burn-Rate (`burn`) Segment — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an opt-in coralline statusline segment, `burn`, that shows the projected time-to-100% for whichever rate-limit window (5h or 7d) binds first — a fuel-gauge "range to empty" like `↗5h ⇢1h58m`.

**Architecture:** A zero-fork sampler inside `statusline.sh` appends `(epoch, 5h%, 5h-reset)` to `~/.claude/coralline/burn-5h.tsv` each render when `VL_BURN=1`. `seg_burn` derives an ETA two ways — 5h from the **recent slope** of the sampled file (1%-crossing based, the value is stepwise), 7d from a **stateless average** of the live `wd_pct`/`wd_rst` inputs — and renders the nearer (binding) one, coloured by that limit's own reset margin.

**Tech Stack:** Bash 3.2+ (macOS) / 4+ (Linux, Git Bash), `awk` (BSD/one-true-awk compatible — **no gawk-isms**: no `asort`/`asorti`/`gensub`), `jq` (already used for parsing). No new dependencies.

## Global Constraints

- **Fork frugality.** The sampler must add **zero** subprocesses per render (`printf … >> file` is a builtin + redirection). The reader (`seg_burn`) may use **one** `awk` pass plus an optional `mv` for trimming; the `%/rate` `awk` call is allowed only when `VL_BURN_SHOWRATE=1`.
- **Default is zero-side-effect.** With `VL_BURN=0` (the default) **nothing is written to disk** and no `burn` segment renders. coralline must stay 100% stateless by default.
- **Opt-in to appear.** `burn` only renders when `VL_BURN=1` *and* `burn` is listed in `VL_SEGMENTS`/`2`/`3`. Dispatch is automatic: `build_segments` calls `seg_burn` if the function exists (statusline.sh:450). No registry edit needed.
- **POSIX-awk only.** Target macOS's default `awk`. Do all sorting/dedup with plain associative arrays and explicit loops.
- **Plain-Unicode glyphs only.** Use geometric Unicode (`↗`, `⇢`, `—`, `…`), never Nerd-Font PUA, matching the existing segment glyphs (`⬡ ⎇ ◆`). These are kept verbatim in `VL_ASCII=1` mode (that mode only swaps caps and bar fill, statusline.sh:72-75), so no per-glyph ASCII branch is needed.
- **Theme colours via semantic vars only.** Colour with `VL_FG_OK/WARN/HOT/DIM`; never hardcode a colour code. Background uses `${VL_BG_BURN:-$VL_BG_5H}` (the use-site fallback pattern from `seg_project`, statusline.sh:331) so it inherits each theme's 5h background.
- **`bash -n statusline.sh` must pass** after every task.

**Function-extraction constraint (affects all awk):** the unit tests pull a single function out of `statusline.sh` with `sed -n '/^name() {/,/^}/p'`. That `sed` stops at the **first line beginning with `}` in column 0**. Therefore every `awk` program embedded in these functions MUST keep all its `{`/`}` **indented** — no brace in column 0 except the function's own closing brace.

---

## File Structure

- **Modify `statusline.sh`:**
  - Defaults block (~line 39 / ~line 66): add `VL_BURN`, `CORALLINE_BURN_WINDOW`, `VL_BURN_SHOWRATE`, `VL_BURN_GLYPH`, `VL_BG_BURN`, `VL_BURN_TRIM`, and `BURN_FILE`.
  - New functions near the other `seg_*`/`fmt_*` helpers: `fmt_eta`, `burn_sample`, `burn_eta_5h`, `burn_eta_7d`, `burn_estimate`, `seg_burn`.
  - Sampler hook: one line at statusline.sh:492 (just before `if [ "$VL_LAYOUT" = "auto" ]`).
- **Create `test/test-burn.sh`** — fixture-driven unit tests, run with `bash test/test-burn.sh`.
- **Modify `README.md` and `README.zh-TW.md`** — segment table row + config keys.

Each task ends with `bash -n statusline.sh` green and (from Task 2 on) `bash test/test-burn.sh` green.

---

## Task 1: Config defaults + sampler

**Files:**
- Modify: `statusline.sh` (defaults block ~39–66; new `fmt_eta`/`burn_sample` near `fmt_duration` ~line 189; hook at ~line 492)
- Test: `test/test-burn.sh` (create)

**Interfaces:**
- Produces:
  - Globals (defaults, user-overridable via `coralline.conf`): `VL_BURN=0`, `CORALLINE_BURN_WINDOW=600`, `VL_BURN_SHOWRATE=0`, `VL_BURN_GLYPH="↗"`, `VL_BG_BURN=""`, `VL_BURN_TRIM=1500`, `BURN_FILE="${CORALLINE_BURN_FILE:-$HOME/.claude/coralline/burn-5h.tsv}"`.
  - `fmt_eta SECONDS` → sets `_ETA` to `"1d11h"` / `"1h58m"` / `"47m"` (mirrors `fmt_countdown`).
  - `burn_sample NOW PCT RESETS_AT_RAW` → appends `epoch<TAB>pct<TAB>reset_epoch\n` to `$BURN_FILE`. Converts `RESETS_AT_RAW` via existing `to_epoch`. No-op if `PCT` empty.

- [ ] **Step 1: Add the defaults.** In `statusline.sh`, immediately after the line `VL_ASCII=0 ...` (statusline.sh:39), insert:

```bash
# ── Burn-rate segment (range-to-empty); opt-in, default off ──────────────────
VL_BURN=0                       # 1 = sample 5h% each render + enable seg_burn
CORALLINE_BURN_WINDOW=600       # recent-slope lookback for 5h, seconds
VL_BURN_SHOWRATE=0              # 1 = also show the binding limit's rate
VL_BURN_GLYPH="↗"               # plain-Unicode, arrow family (kept in VL_ASCII)
VL_BG_BURN=""                   # empty → inherits VL_BG_5H at the use site
VL_BURN_TRIM=1500               # max rows kept in the sample file
BURN_FILE="${CORALLINE_BURN_FILE:-$HOME/.claude/coralline/burn-5h.tsv}"
```

- [ ] **Step 2: Write the failing test (sampler + fmt_eta).** Create `test/test-burn.sh`:

```bash
#!/usr/bin/env bash
# Unit tests for the burn-rate segment helpers. Each function is pulled live
# from statusline.sh so the tests can never drift from the implementation.
#   bash test/test-burn.sh
set -u
HERE=$(cd "$(dirname "$0")" && pwd)
SCRIPT="$HERE/../statusline.sh"
TMPD=$(mktemp -d)
trap 'rm -rf "$TMPD"' EXIT
fail=0
ok()   { printf 'ok    %s\n' "$1"; }
bad()  { printf 'FAIL  %s — %s\n' "$1" "$2"; fail=1; }
eq()   { [ "$2" = "$3" ] && ok "$1" || bad "$1" "want=$3 got=$2"; }

# Pull the helpers under test out of the real script.
eval "$(sed -n '/^to_epoch() {/,/^}/p'     "$SCRIPT")"
eval "$(sed -n '/^fmt_eta() {/,/^}/p'       "$SCRIPT")"
eval "$(sed -n '/^burn_sample() {/,/^}/p'   "$SCRIPT")"

# fmt_eta
fmt_eta 0;       eq "fmt_eta 0m"     "$_ETA" "0m"
fmt_eta 2820;    eq "fmt_eta 47m"    "$_ETA" "47m"
fmt_eta 7080;    eq "fmt_eta 1h58m"  "$_ETA" "1h58m"
fmt_eta 127800;  eq "fmt_eta 1d11h"  "$_ETA" "1d11h"

# burn_sample appends one row with the reset converted to epoch
BURN_FILE="$TMPD/burn.tsv"
burn_sample 1781794590 6 1781811000
eq "sample row" "$(cat "$BURN_FILE")" "$(printf '1781794590\t6\t1781811000')"

# empty pct → no-op (file unchanged)
burn_sample 1781794600 "" 1781811000
eq "sample empty-pct no-op" "$(wc -l < "$BURN_FILE" | tr -d ' ')" "1"

[ "$fail" -eq 0 ] && echo "ALL PASS" || { echo "SOME FAILED"; exit 1; }
```

- [ ] **Step 3: Run the test to confirm it fails.**

Run: `bash test/test-burn.sh`
Expected: FAIL — `fmt_eta`/`burn_sample` not defined (the `eval`'d sed output is empty).

- [ ] **Step 4: Implement `fmt_eta` and `burn_sample`.** In `statusline.sh`, immediately after the `fmt_duration() { … }` function (ends statusline.sh:189), insert. **Keep every awk/inner brace indented — no column-0 `}` except each function's own closer.**

```bash
fmt_eta() {  # → _ETA ; $1=seconds (mirrors fmt_countdown's d/h/m formatting)
  local s="${1:-0}" d h m
  d=$(( s / 86400 )); h=$(( (s % 86400) / 3600 )); m=$(( (s % 3600) / 60 ))
  if   [ "$d" -gt 0 ]; then printf -v _ETA '%dd%02dh' "$d" "$h"
  elif [ "$h" -gt 0 ]; then printf -v _ETA '%dh%02dm' "$h" "$m"
  else                      printf -v _ETA '%dm' "$m"; fi
}

burn_sample() {  # append one 5h sample; $1=now $2=pct(raw) $3=resets_at(raw)
  [ -n "$2" ] || return 0
  to_epoch "$3" || return 0
  printf '%s\t%s\t%s\n' "$1" "$2" "$_EP" >> "$BURN_FILE" 2>/dev/null
}
```

- [ ] **Step 5: Add the sampler hook.** In `statusline.sh`, on the blank line just before `if [ "$VL_LAYOUT" = "auto" ]; then` (statusline.sh:492), insert:

```bash
[ "$VL_BURN" = "1" ] && burn_sample "$NOW" "$fh_pct" "$fh_rst"
```

- [ ] **Step 6: Run the test and the syntax check.**

Run: `bash -n statusline.sh && bash test/test-burn.sh`
Expected: `bash -n` silent (exit 0); test prints `ALL PASS`.

- [ ] **Step 7: Commit.**

```bash
git add statusline.sh test/test-burn.sh
git commit -m "feat(burn): config defaults, fmt_eta, and the 5h sampler"
```

---

## Task 2: 5h recent-slope estimator (`burn_eta_5h`)

**Files:**
- Modify: `statusline.sh` (add `burn_eta_5h` after `burn_sample`)
- Test: `test/test-burn.sh` (extend)

**Interfaces:**
- Consumes: globals `BURN_FILE`, `NOW`, `CORALLINE_BURN_WINDOW`, `VL_BURN_TRIM`; the TSV format from Task 1.
- Produces: `burn_eta_5h` → sets `_B5_STATE` (`active`|`idle`|`warming`), `_B5_ETA` (integer seconds, or the string `inf`), `_B5_RATE` (%/sec as a decimal string, or `0`), `_B5_TTR` (integer seconds until the 5h reset, or `0`). Side effect: trims `$BURN_FILE` in place to the last `VL_BURN_TRIM` rows when it is longer.

**Algorithm (single awk pass, file order — no sort):** dedup by epoch (last wins); detect a window reset as an integer-`%` *decrease* and restart accounting after it; a 1% *crossing* is a sample whose integer `%` exceeds its predecessor's, timestamped at the new level; the recent rate uses the first and last crossings that fall inside `[NOW-window, NOW]`. `active` needs ≥2 in-window crossings; if crossings exist only outside the window → `idle`; otherwise `warming`. (Cross-session interleaving can momentarily disorder rows; it self-heals next render — an accepted simplification in lieu of a sort fork.)

- [ ] **Step 1: Write the failing tests.** Append to `test/test-burn.sh`, before the final pass/fail line:

```bash
eval "$(sed -n '/^burn_eta_5h() {/,/^}/p' "$SCRIPT")"
CORALLINE_BURN_WINDOW=600
VL_BURN_TRIM=1500

# helper: write a fixture and run the estimator at a given "now"
run5h() { BURN_FILE="$TMPD/b5.tsv"; printf '%b' "$1" > "$BURN_FILE"; NOW="$2"; burn_eta_5h; }

# active: 6→7 at +60s, 7→8 at +300s; now=+360s; reset 4h25m out.
# crossings in window: (60,7),(300,8) → rate=(8-7)/(300-60)=1/240 %/s
# now pct=8 → ETA=(100-8)/(1/240)=22080s=6h08m
RST=$(( 1000000 + 18000 ))     # window opened at t=1000000-? use reset far ahead
run5h "1000000\t6\t1015900\n1000060\t7\t1015900\n1000300\t8\t1015900\n1000360\t8\t1015900\n" 1000360
eq "5h active state" "$_B5_STATE" "active"
eq "5h active eta"   "$_B5_ETA"   "22080"
eq "5h ttr"          "$_B5_TTR"   "15540"

# idle: only crossing is older than the 600s window (at +0s); now=+1200s
run5h "1000000\t6\t1015900\n1000010\t7\t1015900\n1001200\t7\t1015900\n" 1001200
eq "5h idle state" "$_B5_STATE" "idle"
eq "5h idle eta"   "$_B5_ETA"   "inf"

# warming: a single crossing, in window
run5h "1000000\t6\t1015900\n1000060\t7\t1015900\n" 1000100
eq "5h warming state" "$_B5_STATE" "warming"
eq "5h warming eta"   "$_B5_ETA"   "inf"

# reset: pct drops mid-file → pre-drop discarded, then only one crossing → warming
run5h "1000000\t80\t1004000\n1000060\t81\t1004000\n1000120\t1\t1019000\n1000180\t2\t1019000\n" 1000200
eq "5h reset→warming" "$_B5_STATE" "warming"

# empty file → warming/inf
run5h "" 1000000
eq "5h empty state" "$_B5_STATE" "warming"
eq "5h empty eta"   "$_B5_ETA"   "inf"

# trim: 5 rows, trim=3 → file keeps last 3
VL_BURN_TRIM=3
run5h "1\t6\t9\n2\t6\t9\n3\t7\t9\n4\t7\t9\n5\t8\t9\n" 6
eq "5h trim rowcount" "$(wc -l < "$TMPD/b5.tsv" | tr -d ' ')" "3"
eq "5h trim first-kept" "$(head -1 "$TMPD/b5.tsv" | cut -f1)" "3"
VL_BURN_TRIM=1500
```

- [ ] **Step 2: Run to confirm failure.**

Run: `bash test/test-burn.sh`
Expected: FAIL — `burn_eta_5h` not defined.

- [ ] **Step 3: Implement `burn_eta_5h`.** In `statusline.sh`, after `burn_sample`, insert (note: all awk braces indented):

```bash
burn_eta_5h() {  # → _B5_STATE _B5_ETA _B5_RATE _B5_TTR ; trims $BURN_FILE
  _B5_STATE="warming"; _B5_ETA="inf"; _B5_RATE="0"; _B5_TTR="0"
  [ -f "$BURN_FILE" ] || return 0
  local tmp="$BURN_FILE.tmp" out
  out=$(awk -F'\t' -v now="$NOW" -v win="$CORALLINE_BURN_WINDOW" \
            -v trim="$VL_BURN_TRIM" -v tmp="$tmp" '
    $2 != "" {
      e = $1 + 0
      if (!(e in seen)) { ord[++n] = e; seen[e] = 1 }
      pct[e] = $2 + 0; rst[e] = $3 + 0
    }
    END {
      if (n == 0) { print "warming inf 0 0"; next_done = 1 }
      if (!next_done) {
        start = 1
        for (i = 2; i <= n; i++)
          if (int(pct[ord[i]]) < int(pct[ord[i-1]])) start = i
        le = ord[n]; lp = pct[le]
        ttr = rst[le] - now; if (ttr < 0) ttr = 0
        cwin = now - win
        fc_t = 0; fc_p = -1; lc_t = 0; lc_p = -1; ncross = 0; anycross = 0
        for (i = start + 1; i <= n; i++) {
          a = int(pct[ord[i-1]]); b = int(pct[ord[i]])
          if (b > a) {
            anycross = 1; ct = ord[i]
            if (ct >= cwin && ct <= now) {
              if (fc_p < 0) { fc_t = ct; fc_p = b }
              lc_t = ct; lc_p = b; ncross++
            }
          }
        }
        if (ncross >= 2 && lc_t > fc_t && lc_p > fc_p) {
          rate = (lc_p - fc_p) / (lc_t - fc_t)
          eta = (100 - lp) / rate; if (eta < 0) eta = 0
          printf "active %.0f %.10f %d\n", eta, rate, ttr
        } else if (anycross && ncross == 0) {
          print "idle inf 0 " ttr
        } else {
          print "warming inf 0 " ttr
        }
        if (n > trim) {
          lo = n - trim + 1
          for (i = lo; i <= n; i++)
            printf "%d\t%s\t%d\n", ord[i], pct[ord[i]], rst[ord[i]] > tmp
        }
      }
    }
  ' "$BURN_FILE")
  [ -f "$tmp" ] && mv "$tmp" "$BURN_FILE" 2>/dev/null
  read -r _B5_STATE _B5_ETA _B5_RATE _B5_TTR <<EOF
$out
EOF
}
```

- [ ] **Step 4: Run the tests.**

Run: `bash -n statusline.sh && bash test/test-burn.sh`
Expected: `ALL PASS`. (If `next_done` causes issues on macOS awk, note `next` inside `END` is illegal in POSIX awk — see Step 5.)

- [ ] **Step 5: Fix the `END`/`next` portability bug if Step 4 fails.** POSIX awk forbids `next` inside `END`. Replace the `END { if (n==0) {…; next_done=1} if(!next_done){…} }` structure with an early-guarded block:

```awk
    END {
      if (n == 0) { print "warming inf 0 0" }
      else {
        start = 1
        # … the entire body from "for (i = 2 …" through the trim loop …
      }
    }
```

Re-run: `bash test/test-burn.sh` → `ALL PASS`.

- [ ] **Step 6: Commit.**

```bash
git add statusline.sh test/test-burn.sh
git commit -m "feat(burn): 5h recent-slope estimator with crossing detection + trim"
```

---

## Task 3: 7d stateless-average estimator (`burn_eta_7d`)

**Files:**
- Modify: `statusline.sh` (add `burn_eta_7d` after `burn_eta_5h`)
- Test: `test/test-burn.sh` (extend)

**Interfaces:**
- Consumes: globals `wd_pct`, `wd_rst` (parsed at statusline.sh:104-105), `NOW`.
- Produces: `burn_eta_7d` → sets `_B7_ETA` (integer seconds or `inf`), `_B7_RATE` (%/sec or `0`), `_B7_TTR` (integer seconds to 7d reset or `0`). No file I/O.

**Math:** `window_start = wd_rst − 7·86400`; `elapsed = NOW − window_start`; `rate = wd_pct / elapsed`; `ETA = (100 − wd_pct) / rate`. `inf` when `wd_pct` empty/≤0 or `elapsed ≤ 0`. `wd_rst` is converted via `to_epoch` first (Claude sends epoch ints — the fork-free path).

- [ ] **Step 1: Write the failing tests.** Append to `test/test-burn.sh`:

```bash
eval "$(sed -n '/^burn_eta_7d() {/,/^}/p' "$SCRIPT")"

# 7d: used 30%, window opened 3 days ago (elapsed=259200s), reset 4 days out.
# rate=30/259200 %/s; ETA=(100-30)/rate=70*259200/30=604800s=7d00h
WS=$(( 1000000 - 259200 )); R7=$(( WS + 604800 ))
wd_pct=30; wd_rst=$R7; NOW=1000000; burn_eta_7d
eq "7d eta"  "$_B7_ETA" "604800"
eq "7d ttr"  "$_B7_TTR" "345600"

# 7d unused → inf
wd_pct=0; wd_rst=$R7; NOW=1000000; burn_eta_7d
eq "7d unused eta" "$_B7_ETA" "inf"

# 7d not reported → inf
wd_pct=""; wd_rst=""; NOW=1000000; burn_eta_7d
eq "7d empty eta" "$_B7_ETA" "inf"
```

- [ ] **Step 2: Run to confirm failure.**

Run: `bash test/test-burn.sh`
Expected: FAIL — `burn_eta_7d` not defined.

- [ ] **Step 3: Implement `burn_eta_7d`.** In `statusline.sh`, after `burn_eta_5h`, insert (awk braces indented):

```bash
burn_eta_7d() {  # → _B7_ETA _B7_RATE _B7_TTR (stateless; uses wd_pct/wd_rst)
  _B7_ETA="inf"; _B7_RATE="0"; _B7_TTR="0"
  [ -n "$wd_pct" ] || return 0
  to_epoch "$wd_rst" || return 0
  read -r _B7_ETA _B7_RATE _B7_TTR <<EOF
$(awk -v p="$wd_pct" -v r="$_EP" -v now="$NOW" 'BEGIN {
    ttr = r - now; if (ttr < 0) ttr = 0
    ws = r - 7 * 86400; el = now - ws
    if (p + 0 <= 0 || el <= 0) { print "inf 0 " ttr; exit }
    rate = (p + 0) / el
    eta = (100 - (p + 0)) / rate; if (eta < 0) eta = 0
    printf "%.0f %.10f %d\n", eta, rate, ttr
  }')
EOF
}
```

- [ ] **Step 4: Run the tests.**

Run: `bash -n statusline.sh && bash test/test-burn.sh`
Expected: `ALL PASS`.

- [ ] **Step 5: Commit.**

```bash
git add statusline.sh test/test-burn.sh
git commit -m "feat(burn): 7d stateless-average estimator"
```

---

## Task 4: Binding selection + `seg_burn` render

**Files:**
- Modify: `statusline.sh` (add `burn_estimate` then `seg_burn` after `burn_eta_7d`)
- Test: `test/test-burn.sh` (extend)

**Interfaces:**
- Consumes: `burn_eta_5h`, `burn_eta_7d` and their `_B5_*`/`_B7_*` outputs; render helpers `fg`, `push`, `fmt_eta`; colour vars `VL_FG_OK/WARN/HOT/DIM`; `VL_BURN_GLYPH`, `VL_BURN_SHOWRATE`, `VL_BG_BURN`, `VL_BG_5H`.
- Produces:
  - `burn_estimate` → sets `_BURN_STATE` (`active`|`idle`|`warming`), `_BURN_LABEL` (`5h`|`7d`|``), `_BURN_ETA` (int sec or `inf`), `_BURN_RATE` (%/sec or `0`), `_BURN_TTR` (int sec). Binding = the finite ETA that is smaller; ties resolve to `5h`.
  - `seg_burn` → `push`es one rendered segment. Active: `↗<5h|7d> ⇢<eta>` coloured by `_BURN_TTR/_BURN_ETA`. Idle: `↗ ⇢—` dim. Warming: `↗ ⇢…` dim.

**Colour rule (integer math, no fork):** `HOT` when `eta ≤ ttr` (ratio ≥ 1); else `WARN` when `10·ttr ≥ 8·eta` (ratio ≥ 0.8); else `OK`.

- [ ] **Step 1: Write the failing tests.** Append to `test/test-burn.sh`:

```bash
eval "$(sed -n '/^fg() {/,/^}/p'            "$SCRIPT")"
eval "$(sed -n '/^push() {/,/^}/p'          "$SCRIPT")"
eval "$(sed -n '/^burn_estimate() {/,/^}/p' "$SCRIPT")"
eval "$(sed -n '/^seg_burn() {/,/^}/p'      "$SCRIPT")"
VL_BURN_GLYPH="↗"; VL_BURN_SHOWRATE=0; VL_BG_BURN=""; VL_BG_5H=237; VL_LAYOUT="fixed"
VL_FG_OK=114; VL_FG_WARN=179; VL_FG_HOT=167; VL_FG_DIM=245

# stub the two estimators so binding logic is tested in isolation
mk5h() { _B5_STATE="$1"; _B5_ETA="$2"; _B5_RATE="$3"; _B5_TTR="$4"; }
mk7d() {                 _B7_ETA="$1"; _B7_RATE="$2"; _B7_TTR="$3"; }
burn_eta_5h() { mk5h "$M5S" "$M5E" "$M5R" "$M5T"; }
burn_eta_7d() { mk7d "$M7E" "$M7R" "$M7T"; }

# 5h roomy (eta 6h), 7d binding (eta 2h) → label 7d
M5S=active M5E=21600 M5R=0 M5T=15000  M7E=7200 M7R=0 M7T=86400
burn_estimate
eq "binding label 7d"  "$_BURN_LABEL" "7d"
eq "binding eta 7d"    "$_BURN_ETA"   "7200"

# 5h binding (eta 1h) vs 7d (eta 10h) → label 5h
M5S=active M5E=3600 M5R=0 M5T=9000  M7E=36000 M7R=0 M7T=200000
burn_estimate
eq "binding label 5h"  "$_BURN_LABEL" "5h"

# 5h idle + 7d unused → idle, no label
M5S=idle M5E=inf M5R=0 M5T=0  M7E=inf M7R=0 M7T=0
burn_estimate
eq "binding idle"      "$_BURN_STATE" "idle"
eq "binding idle nolabel" "$_BURN_LABEL" ""

# render: active 7d binding, eta 2h, ttr 1h → ratio ttr/eta=0.5 (<0.8) → OK colour;
# 5h roomy (eta 6h) so 7d wins. Contains ↗7d and ⇢2h00m.
SEG_BGS=(); SEG_TXT=(); SEG_LEN=()
M5S=active M5E=21600 M5R=0 M5T=15000  M7E=7200 M7R=0 M7T=3600
seg_burn
case "${SEG_TXT[0]}" in *"↗7d"*"⇢2h00m"*) ok "render active 7d" ;; *) bad "render active 7d" "got=${SEG_TXT[0]}" ;; esac
case "${SEG_TXT[0]}" in *$'\033[38;5;114m'*) ok "render OK colour" ;; *) bad "render OK colour" "no OK fg in ${SEG_TXT[0]}" ;; esac

# render: active 5h binding, eta 5m ≤ ttr 10m → you empty before reset → HOT colour
SEG_BGS=(); SEG_TXT=(); SEG_LEN=()
M5S=active M5E=300 M5R=0 M5T=600  M7E=inf M7R=0 M7T=0
seg_burn
case "${SEG_TXT[0]}" in *$'\033[38;5;167m'*) ok "render HOT colour" ;; *) bad "render HOT colour" "no HOT fg in ${SEG_TXT[0]}" ;; esac

# render: idle → dim, contains ⇢—
SEG_BGS=(); SEG_TXT=(); SEG_LEN=()
M5S=idle M5E=inf M5R=0 M5T=0  M7E=inf M7R=0 M7T=0
seg_burn
case "${SEG_TXT[0]}" in *"⇢—"*$'\033'*|*$'\033'*"⇢—"*) ok "render idle dash" ;; *) bad "render idle dash" "got=${SEG_TXT[0]}" ;; esac
```

- [ ] **Step 2: Run to confirm failure.**

Run: `bash test/test-burn.sh`
Expected: FAIL — `burn_estimate`/`seg_burn` not defined.

- [ ] **Step 3: Implement `burn_estimate` and `seg_burn`.** In `statusline.sh`, after `burn_eta_7d`, insert:

```bash
burn_estimate() {  # → _BURN_STATE _BURN_LABEL _BURN_ETA _BURN_RATE _BURN_TTR
  burn_eta_5h; burn_eta_7d
  local f5=0 f7=0
  [ "$_B5_ETA" != "inf" ] && f5=1
  [ "$_B7_ETA" != "inf" ] && f7=1
  if [ "$f5" = 1 ] && { [ "$f7" = 0 ] || [ "$_B5_ETA" -le "$_B7_ETA" ]; }; then
    _BURN_STATE="active"; _BURN_LABEL="5h"
    _BURN_ETA="$_B5_ETA"; _BURN_RATE="$_B5_RATE"; _BURN_TTR="$_B5_TTR"
  elif [ "$f7" = 1 ]; then
    _BURN_STATE="active"; _BURN_LABEL="7d"
    _BURN_ETA="$_B7_ETA"; _BURN_RATE="$_B7_RATE"; _BURN_TTR="$_B7_TTR"
  else
    _BURN_ETA="inf"; _BURN_RATE="0"; _BURN_TTR="0"; _BURN_LABEL=""
    if [ "$_B5_STATE" = "idle" ]; then _BURN_STATE="idle"; else _BURN_STATE="warming"; fi
  fi
}

seg_burn() {
  burn_estimate
  local bg="${VL_BG_BURN:-$VL_BG_5H}"
  if [ "$_BURN_STATE" != "active" ]; then
    fg "$VL_FG_DIM"
    if [ "$_BURN_STATE" = "idle" ]; then push "$bg" "${_FG} ${VL_BURN_GLYPH} ⇢— "
    else                                  push "$bg" "${_FG} ${VL_BURN_GLYPH} ⇢… "; fi
    return 0
  fi
  local eta="$_BURN_ETA" ttr="$_BURN_TTR" col rate=""
  if   [ "$eta" -le "$ttr" ];               then col="$VL_FG_HOT"
  elif [ $(( 10 * ttr )) -ge $(( 8 * eta )) ]; then col="$VL_FG_WARN"
  else                                            col="$VL_FG_OK"; fi
  fmt_eta "$eta"
  if [ "$VL_BURN_SHOWRATE" = "1" ]; then
    if [ "$_BURN_LABEL" = "5h" ]; then
      rate=$(awk -v r="$_BURN_RATE" 'BEGIN { printf "%.1f%%/10m ", r * 600 }')
    else
      rate=$(awk -v r="$_BURN_RATE" 'BEGIN { printf "%.0f%%/d ", r * 86400 }')
    fi
  fi
  fg "$col"
  push "$bg" "${_FG} ${VL_BURN_GLYPH}${_BURN_LABEL} ${rate}⇢${_ETA} "
}
```

- [ ] **Step 4: Run the tests.**

Run: `bash -n statusline.sh && bash test/test-burn.sh`
Expected: `ALL PASS`.

- [ ] **Step 5: End-to-end smoke test against the real script.** `VL_BURN`/`VL_SEGMENTS` cannot be passed as env vars — the defaults block (statusline.sh:~40) reassigns them; the only override channel is the config file (sourced at statusline.sh:70). So drive it through a temp conf. Anchor the fixture timestamps to *now* (the recent-slope window is relative to the live clock):

```bash
now=$(date +%s)
printf '%s\t6\t%s\n%s\t7\t%s\n%s\t8\t%s\n' \
  $((now-300)) $((now+15000)) $((now-240)) $((now+15000)) $((now-20)) $((now+15000)) \
  > /tmp/burn-smoke.tsv
cat > /tmp/burn-smoke.conf <<EOF
VL_BURN=1
VL_SEGMENTS=burn
BURN_FILE=/tmp/burn-smoke.tsv
EOF
echo "{\"rate_limits\":{\"five_hour\":{\"used_percentage\":8,\"resets_at\":$((now+15000))},\"seven_day\":{\"used_percentage\":0,\"resets_at\":$((now+200000))}}}" \
  | CORALLINE_CONFIG=/tmp/burn-smoke.conf bash statusline.sh 2>&1 | cat -v | head
```
Expected: output contains `↗5h` and `⇢` (a finite ETA) — two crossings (`6→7` at now-240, `7→8` at now-20) fall inside the 600s window, so state is `active`. Confirms end-to-end wiring; the unit tests pin the exact math.

- [ ] **Step 6: Commit.**

```bash
git add statusline.sh test/test-burn.sh
git commit -m "feat(burn): binding-constraint selection and seg_burn rendering"
```

---

## Task 5: Documentation

**Files:**
- Modify: `README.md` (segment table; config/requirements note)
- Modify: `README.zh-TW.md` (matching rows)

**Interfaces:** none (docs only). configure.sh wizard integration is intentionally **out of scope for v1** — `burn` is enabled by setting `VL_BURN=1` and adding `burn` to `VL_SEGMENTS` in `coralline.conf`, which these docs describe. A future spec can add a wizard toggle.

- [ ] **Step 1: Add the segment-table row in `README.md`.** In the `| Segment | Shows |` table, after the `limit5h` / `limit7d` row, add:

```markdown
| `burn` | range-to-empty: projected time until the binding limit (5h or 7d) hits 100% at the recent burn rate (`↗`); opt-in via `VL_BURN=1` |
```

- [ ] **Step 2: Document the config keys in `README.md`.** Wherever config keys are listed (or in a short new "Burn-rate segment" note near the gauge description), add:

```markdown
**Burn-rate (`burn`) segment.** Off by default. Set `VL_BURN=1` and add `burn` to
`VL_SEGMENTS` to show a "range to empty" — the projected time until whichever rate
limit (5h or 7d) binds first, e.g. `↗5h ⇢1h58m`. It colours green/yellow/red by
whether you'll hit that wall before the window resets. Keys: `CORALLINE_BURN_WINDOW`
(recent-slope lookback, default 600s), `VL_BURN_SHOWRATE` (also show the rate),
`VL_BURN_GLYPH` (default `↗`), `VL_BG_BURN` (defaults to the 5h background),
`VL_BURN_TRIM` (max samples kept, default 1500). When `VL_BURN=1`, coralline writes
samples to `~/.claude/coralline/burn-5h.tsv`; with the default `VL_BURN=0` nothing is
written.
```

- [ ] **Step 3: Mirror both additions in `README.zh-TW.md`** with the equivalent Traditional Chinese rows (same glyph examples and key names).

- [ ] **Step 4: Commit.**

```bash
git add README.md README.zh-TW.md
git commit -m "docs(burn): document the burn-rate segment and its config keys"
```

---

## Self-Review

**Spec coverage:**
- 5h recent-slope estimator → Task 2. 7d stateless average → Task 3. Binding selection (`min(ETA)`, label, tie→5h) → Task 4. Colour ratio on the binding limit's reset → Task 4 (Step 3). Zero-fork sampler gated by `VL_BURN`, 5h-only file → Task 1. `warming`/`idle`/`active`/`reset` state machine + "never freeze a stale ETA" (non-active → `inf`) → Task 2. Integer-quantization handling (crossing-based) → Task 2. Account-global / concurrent append (no sort, self-heals) → Task 2 note. Theme portability (semantic colours, `${VL_BG_BURN:-$VL_BG_5H}`) → Global Constraints + Task 4. Glyph `↗` plain-Unicode → Task 1. Config keys table → Task 1 + Task 5. Edge cases (5h/7d not reported, handoff, both-have-room, clock skew, trim) → covered by Tasks 2-4 logic and tests. `VL_BURN=0` ⇒ no file/segment → Global Constraints + Task 1 hook.
- **Deferred from spec:** configure.sh wizard toggle (spec "Config/installer integration") — explicitly scoped out of v1 in Task 5; documented enable path instead.

**Placeholder scan:** every code step contains the actual bash/awk/markdown; no TBD/TODO; test bodies are concrete with expected values.

**Type/name consistency:** globals `_B5_STATE/_B5_ETA/_B5_RATE/_B5_TTR`, `_B7_ETA/_B7_RATE/_B7_TTR`, `_BURN_STATE/_BURN_LABEL/_BURN_ETA/_BURN_RATE/_BURN_TTR`, `_ETA`, `BURN_FILE`, `VL_BURN*` are used identically across Tasks 1-5. `fmt_eta` sets `_ETA`; `seg_burn` reads `_ETA` — consistent. Sentinel `inf` is produced by both estimators and tested for in `burn_estimate`.
