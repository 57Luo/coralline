# docs-accuracy Specification

## Purpose

TBD - created by archiving change 'docs-truth-sync'. Update Purpose after archive.

## Requirements

### Requirement: Segment support tables state actual renderer coverage

The segment compatibility tables in README.md, README.zh-TW.md, and INSTALL.md SHALL state that the Go renderer supports all 18 segments, and SHALL list only the lean/classic styles and auto layout as bash-only features.

#### Scenario: Reader checks whether a segment requires the bash renderer

- **WHEN** a reader consults any of the three documents for segment support
- **THEN** every segment implemented in the Go renderer's builder registry is documented as Go-supported, with no segment falsely marked bash-only

##### Example: segments previously mislabeled

| Segment | Old table said | Corrected table says |
| ------- | -------------- | -------------------- |
| cost, clock, lines, style, duration, stash, project, node, python | bash-only ("—") | Go supported |
| lean/classic styles, auto layout | bash-only | bash-only (unchanged) |


<!-- @trace
source: docs-truth-sync
updated: 2026-07-16
code:
  - INSTALL.md
  - README.md
  - UPGRADE.md
  - README.zh-TW.md
-->

---
### Requirement: Install and upgrade commands point at this repository

All installation and upgrade commands in INSTALL.md and UPGRADE.md SHALL reference the 57Luo/coralline repository. No command SHALL reference the retired upstream repository (Nanako0129/coralline).

#### Scenario: User follows the quick-install command

- **WHEN** a user copies the curl install or upgrade command from the documentation
- **THEN** the command fetches from 57Luo/coralline and installs the version that includes the Go renderer


<!-- @trace
source: docs-truth-sync
updated: 2026-07-16
code:
  - INSTALL.md
  - README.md
  - UPGRADE.md
  - README.zh-TW.md
-->

---
### Requirement: Upgrade documentation covers Go binary users

UPGRADE.md SHALL document the upgrade path for users running the Go binary, including when a rebuild of the binary is required and that existing configuration files remain compatible.

#### Scenario: Go binary user consults upgrade docs after pulling changes

- **WHEN** a user who runs the Go binary reads UPGRADE.md after updating the repository
- **THEN** the document states how to rebuild the binary and whether their existing configuration continues to work


<!-- @trace
source: docs-truth-sync
updated: 2026-07-16
code:
  - INSTALL.md
  - README.md
  - UPGRADE.md
  - README.zh-TW.md
-->

---
### Requirement: Canonical specs carry a filled Purpose section

The canonical specs openspec/specs/go-renderer-segments/spec.md and openspec/specs/limit-snapshot-reset-detection/spec.md SHALL contain a Purpose section describing the capability's scope in one to three sentences, with no TBD placeholder text.

#### Scenario: Spec reader checks capability scope

- **WHEN** a reader opens either canonical spec
- **THEN** the Purpose section describes what the capability covers and contains no "TBD" text

<!-- @trace
source: docs-truth-sync
updated: 2026-07-16
code:
  - INSTALL.md
  - README.md
  - UPGRADE.md
  - README.zh-TW.md
-->