## ADDED Requirements

### Requirement: Render statusline from stdin session JSON

The Go renderer SHALL read a Claude Code session JSON document from stdin and write a pill-style statusline to stdout, then exit. It SHALL extract exactly these fields, treating any missing field as empty/zero: workspace.current_dir (fallback: cwd), model.display_name, context_window.used_percentage, context_window.total_input_tokens, context_window.total_output_tokens, context_window.current_usage.cache_read_input_tokens, context_window.current_usage.cache_creation_input_tokens, rate_limits.five_hour.used_percentage, rate_limits.five_hour.resets_at, rate_limits.seven_day.used_percentage, rate_limits.seven_day.resets_at, and effort.level. Scalar fields whose JSON type varies by producer — used_percentage and resets_at values in particular — SHALL be accepted as either JSON string or JSON number and captured as their raw text (equivalent to the bash implementation's jq `tostring`); a type variation in one field MUST NOT degrade any other field. Malformed JSON MUST degrade to all-empty fields; the renderer MUST NOT panic and MUST NOT write error text to stdout.

#### Scenario: Valid session JSON renders a statusline

- **WHEN** a valid session JSON document is piped to stdin
- **THEN** the renderer writes at least one line of ANSI-colored statusline text to stdout and exits with code 0

#### Scenario: Numeric resets_at is accepted

- **WHEN** rate_limits.five_hour.resets_at arrives as the JSON number 1784028600 instead of an ISO string
- **THEN** all fields parse normally and the five-hour reset is taken from the numeric epoch

##### Example: live Claude Code payload types

- **GIVEN** stdin `{"model":{"display_name":"X"},"context_window":{"used_percentage":19},"rate_limits":{"five_hour":{"used_percentage":92,"resets_at":1784028600}}}`
- **WHEN** the input is parsed
- **THEN** Model="X", CtxPct="19", FhPct="92", FhRst="1784028600" (no field degrades)

#### Scenario: Malformed JSON degrades silently

- **WHEN** stdin contains text that is not valid JSON
- **THEN** the renderer exits with code 0 and stdout contains no error message text

### Requirement: Configuration file compatibility

The renderer SHALL read the same configuration file as the bash implementation (default `~/.claude/coralline.conf`, path overridable via the `VL_CONFIG` environment variable). The parser SHALL support: comment lines and blank lines (skipped), `VAR=value` and `VAR="value"` and `VAR='value'` assignments, and theme-source lines of the form `. "$_VL_DIR/themes/<name>.conf"` which SHALL be expanded by parsing the referenced theme file in place, resolving `$_VL_DIR` to the directory containing the renderer executable (matching the bash implementation, where `_VL_DIR` is the directory of statusline.sh itself — in the real deployment `~/.claude/coralline`, one level below the config file's directory `~/.claude`). Lines outside this subset SHALL be silently ignored. Precedence SHALL be: built-in defaults first, then file assignments in order (later assignments overwrite earlier ones), matching bash source semantics.

#### Scenario: User config with theme source is honored

- **WHEN** the config file assigns VL_STYLE="pill" and sources a theme file that assigns segment colors
- **THEN** the rendered output uses the pill style and the theme's colors

##### Example: real deployment layout

- **GIVEN** config at `~/.claude/coralline.conf` sourcing `. "$_VL_DIR/themes/catppuccin-mocha.conf"`, the renderer executable at `~/.claude/coralline/coralline.exe`, and the theme at `~/.claude/coralline/themes/catppuccin-mocha.conf`
- **WHEN** the config is loaded
- **THEN** the theme file is found and its color assignments take effect (identical colors to the bash implementation sourcing the same files)

#### Scenario: Unsupported syntax lines are ignored

- **WHEN** the config file contains a line that is not a supported assignment, comment, or theme-source line
- **THEN** the renderer skips that line and rendering proceeds using the remaining assignments

### Requirement: Core segment set

The renderer SHALL implement these eight segments with the same visibility rules and content as the bash reference implementation: `dir` (current directory with long paths collapsed), `git` (branch, staged `+` / modified `!` / untracked `?` marks, ahead/behind counts), `model` (model display name), `effort` (reasoning effort level), `ctx` (context-window gauge; token detail suppressed when VL_CTX_TOKENS=0), `limit5h` and `limit7d` (rate-limit gauges with reset countdown), and `burn` (range-to-empty projection). A segment whose source data is empty SHALL be hidden. Segment names present in VL_SEGMENTS lists but not implemented in this change SHALL be silently skipped.

#### Scenario: Git segment hidden outside a repository

- **WHEN** workspace.current_dir points to a directory that is not inside a git repository
- **THEN** the git segment does not appear in the output

#### Scenario: Gauge coloring follows thresholds

- **WHEN** a gauge percentage crosses VL_WARN_PCT or VL_HOT_PCT
- **THEN** the gauge color changes to the warning or hot color respectively

##### Example: threshold boundaries with VL_WARN_PCT=50 VL_HOT_PCT=80

| Percentage | Color state |
| ---------- | ----------- |
| 49         | normal      |
| 50         | warning     |
| 79         | warning     |
| 80         | hot         |

### Requirement: Fixed multi-line layout

The renderer SHALL place segments on up to VL_MAX_LINES lines according to the VL_SEGMENTS, VL_SEGMENTS2, and VL_SEGMENTS3 lists (one list per line, in order). A line whose segments are all hidden SHALL be omitted entirely. Token count abbreviation SHALL match the bash implementation: values >= 1000 render as `<n>.<d>k` and values >= 1000000 render as `<n>.<d>M` using integer arithmetic.

#### Scenario: Three configured lines render in order

- **WHEN** VL_SEGMENTS, VL_SEGMENTS2, and VL_SEGMENTS3 each list at least one visible segment
- **THEN** stdout contains three lines, each rendering its own list's segments in list order

##### Example: token abbreviation

| Input   | Output |
| ------- | ------ |
| 999     | 999    |
| 1234    | 1.2k   |
| 1234567 | 1.2M   |

#### Scenario: Gauge bar fill is rounded

- **WHEN** a gauge bar of width 5 renders a percentage
- **THEN** the number of filled cells equals (pct * 5 + 50) / 100 using integer division, capped at 5
