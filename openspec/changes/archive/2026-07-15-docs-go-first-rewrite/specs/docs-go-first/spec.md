## ADDED Requirements

### Requirement: Documentation presents the Go renderer as the primary path

README.md, README.zh-TW.md, and INSTALL.md SHALL present the Go renderer (native .exe) as the primary install and usage path, and SHALL state the fork's rationale (Windows MSYS zombie-process incident, hot path rewritten in Go) with a reference to openspec/changes/go-renderer-core/proposal.md.

#### Scenario: Reader opens the README

- **WHEN** a reader opens README.md or README.zh-TW.md
- **THEN** the opening section states this fork's Go hot-path rewrite and its rationale before any bash-version instructions appear

#### Scenario: Reader follows the install guide

- **WHEN** a reader follows INSTALL.md from the top
- **THEN** the first install path documented is building the Go renderer (go build) and registering the compiled .exe as the Claude Code statusLine command

### Requirement: Go coverage is stated honestly

The documentation SHALL state that the Go renderer currently covers exactly 8 segments (ctx, git, dir, model, effort, limit5h, limit7d, burn), the pill style, and the fixed multi-line layout, and SHALL NOT claim availability of unported segments or of the lean/classic styles under the Go renderer.

#### Scenario: Segment table shows Go support status

- **WHEN** a reader views the segment table in either README
- **THEN** all 16 segments remain listed and each row indicates whether the Go renderer supports it

#### Scenario: Reader needs an unported feature

- **WHEN** a reader needs a segment or style the Go renderer does not cover
- **THEN** the documentation directs them to the bash version as the fallback path

### Requirement: Bash version is retained as an appendix

The documentation SHALL retain the bash-version install and usage instructions in a clearly-labeled appendix or compatibility section, and SHALL state that both renderers share the same coralline.conf and themes files.

#### Scenario: Reader installs the bash version

- **WHEN** a reader follows the bash appendix in INSTALL.md
- **THEN** the install.sh-based flow is documented completely enough to install without consulting git history

#### Scenario: Reader switches renderers

- **WHEN** a reader switches between the Go and bash renderers
- **THEN** the documentation states that no conf or theme changes are required because both read the same files

### Requirement: The two README languages stay aligned

README.md (English) and README.zh-TW.md (Traditional Chinese) SHALL carry equivalent structure and information after the rewrite; equivalence is structural, not sentence-by-sentence translation.

#### Scenario: Comparing the two READMEs

- **WHEN** the section headings of README.md and README.zh-TW.md are compared
- **THEN** every section present in one language has a counterpart section in the other
