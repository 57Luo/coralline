# go-renderer-segments Specification

## Purpose

Define the segments ported to the Go renderer beyond the original core set (cost, clock, lines, tokens, style, duration, stash, project, node, python), each mirroring the bash renderer's visible output, hide conditions, and configuration knobs so the two renderers stay interchangeable on the same coralline.conf.

## Requirements

### Requirement: cost-segment

The Go renderer SHALL render a `cost` segment displaying the session cost in USD, formatted to the number of decimal places specified by the `VL_COST_DECIMALS` config value. The segment SHALL be silent when the cost value is empty or zero.

#### Scenario: cost segment renders formatted USD

- **WHEN** stdin JSON contains `cost.total_cost_usd` = "1.2345" and config `VL_COST_DECIMALS` = 2
- **THEN** the segment outputs "$1.23"

#### Scenario: cost segment silent on zero

- **WHEN** stdin JSON contains `cost.total_cost_usd` = "0" or is absent
- **THEN** the segment produces no output

---
### Requirement: clock-segment

The Go renderer SHALL render a `clock` segment displaying the current time. The format SHALL be controlled by `VL_CLOCK` config: "24h" for HH:MM, "12h" for hh:mm am/pm, "off" to suppress. `VL_CLOCK_SECONDS` = "1" SHALL append seconds.

#### Scenario: clock 24h format

- **WHEN** config `VL_CLOCK` = "24h" and `VL_CLOCK_SECONDS` is not "1"
- **THEN** the segment outputs time in HH:MM format with the clock glyph

#### Scenario: clock 12h format with lowercase am/pm

- **WHEN** config `VL_CLOCK` = "12h"
- **THEN** the segment outputs time in hh:mm am/pm format with lowercase am/pm

#### Scenario: clock off

- **WHEN** config `VL_CLOCK` = "off"
- **THEN** the segment produces no output

---
### Requirement: lines-segment

The Go renderer SHALL render a `lines` segment displaying lines added and removed. The segment SHALL be silent when both values are zero.

#### Scenario: lines segment renders counts

- **WHEN** stdin JSON contains `cost.total_lines_added` = 42 and `cost.total_lines_removed` = 7
- **THEN** the segment outputs "+42 -7" with OK color for additions and HOT color for removals

#### Scenario: lines segment silent on zero

- **WHEN** both `cost.total_lines_added` and `cost.total_lines_removed` are 0
- **THEN** the segment produces no output

---
### Requirement: tokens-segment

The Go renderer SHALL render a `tokens` segment displaying standalone input/output/cache token counts, using the same fmt_tok formatting as the ctx segment inline tokens.

#### Scenario: tokens segment renders counts

- **WHEN** stdin JSON contains non-zero token counts
- **THEN** the segment outputs token counts with DIM foreground

---
### Requirement: style-segment

The Go renderer SHALL render a `style` segment displaying the active output style name. The segment SHALL be silent when the style is empty or "default".

#### Scenario: style segment renders non-default style

- **WHEN** stdin JSON contains `output_style.name` = "concise"
- **THEN** the segment outputs the style with a pen glyph

#### Scenario: style segment silent on default

- **WHEN** `output_style.name` is "default" or absent
- **THEN** the segment produces no output

---
### Requirement: duration-segment

The Go renderer SHALL render a `duration` segment displaying the session wall-clock duration, formatted from milliseconds to a human-readable string (e.g., "1h 23m", "45s").

#### Scenario: duration segment renders formatted time

- **WHEN** stdin JSON contains `cost.total_duration_ms` = 5025000
- **THEN** the segment outputs the duration with an hourglass glyph in human-readable format matching bash

#### Scenario: duration segment silent on zero

- **WHEN** `cost.total_duration_ms` is 0 or absent
- **THEN** the segment produces no output

---
### Requirement: stash-segment

The Go renderer SHALL render a `stash` segment displaying the git stash count. It SHALL execute `git rev-list --walk-reflogs --count refs/stash` in the working directory. The segment SHALL be silent when not in a git repo or when stash count is zero.

#### Scenario: stash segment renders count

- **WHEN** the working directory is a git repo with 3 stashed entries
- **THEN** the segment outputs the stash glyph followed by "3"

#### Scenario: stash segment silent outside git

- **WHEN** the working directory is not a git repo
- **THEN** the segment produces no output

---
### Requirement: project-segment

The Go renderer SHALL render a `project` segment displaying the git repository root directory name. When not in a git repo, it SHALL fall back to the dir segment behavior, unless dir is already in the segment list. The name SHALL be truncated to VL_NAME_MAX characters.

#### Scenario: project segment in git repo

- **WHEN** the working directory is inside a git repo rooted at /home/user/my-project
- **THEN** the segment outputs the project glyph followed by "my-project"

#### Scenario: project segment falls back to dir

- **WHEN** the working directory is not a git repo and dir is not in the segment list
- **THEN** the segment renders the current directory path (same as dir segment)

---
### Requirement: node-segment

The Go renderer SHALL render a `node` segment displaying the active Node.js version. Detection SHALL walk ancestor directories for .nvmrc or .node-version files. When VL_RUNTIME_PROBE = "1", it SHALL fall back to executing `node --version`. The version string SHALL strip the leading v prefix. The segment SHALL be silent when no version is detected.

#### Scenario: node segment from pin file

- **WHEN** a .nvmrc file containing "v20.11.0" exists in the working directory
- **THEN** the segment outputs the Node glyph followed by "20.11.0"

#### Scenario: node segment silent without pin file or probe

- **WHEN** no .nvmrc or .node-version exists in any ancestor and VL_RUNTIME_PROBE is not "1"
- **THEN** the segment produces no output

---
### Requirement: python-segment

The Go renderer SHALL render a `python` segment displaying the active Python environment. Detection priority: VIRTUAL_ENV env var (basename), CONDA_DEFAULT_ENV env var (skip "base"), .python-version pin file walk. When VL_RUNTIME_PROBE = "1", it SHALL fall back to executing `python3 --version`. The segment SHALL be silent when no environment is detected.

#### Scenario: python segment from VIRTUAL_ENV

- **WHEN** environment variable VIRTUAL_ENV = "/home/user/.venvs/myenv"
- **THEN** the segment outputs the Python glyph followed by "myenv"

#### Scenario: python segment skips conda base

- **WHEN** CONDA_DEFAULT_ENV = "base" and no other detection succeeds
- **THEN** the segment produces no output

---
### Requirement: input-parsing-parity

The Go binary inputjson.Input struct SHALL parse all JSON fields consumed by any of the 18 segments. The new fields SHALL use the same flexStr type for string/number agnostic parsing where applicable.

#### Scenario: new fields parsed from stdin

- **WHEN** stdin JSON contains cost.total_cost_usd, cost.total_lines_added, cost.total_lines_removed, output_style.name, cost.total_duration_ms
- **THEN** the corresponding Input struct fields are populated with the parsed values

---
### Requirement: configure-go-binary-hook

The configure.sh update_settings function SHALL detect the presence of a compiled Go binary (coralline or coralline.exe) in the install directory. When found and executable, the hook command in settings.json SHALL point to the Go binary. When absent, it SHALL fall back to bash statusline.sh.

#### Scenario: Go binary present on install

- **WHEN** the Go binary exists in the install directory and is executable
- **THEN** settings.json statusLine.command points to the Go binary path

#### Scenario: Go binary absent falls back to bash

- **WHEN** no Go binary exists in the install directory
- **THEN** settings.json statusLine.command uses bash statusline.sh as current behavior
