# self-update Specification

## Purpose

Provide a single-command way to update the installed statusline from a repo checkout (update.ps1 / update.sh): pull, run the full test suite, and only on success build and deploy the binary and changed themes. Failed tests or builds must never touch the working installation.

## Requirements

### Requirement: Single-command statusline update

The repository SHALL provide an update script (update.ps1 for Windows, update.sh for POSIX) that, when run from the repository root, updates the installed statusline in this order: pull the current branch, run the full Go test suite, build the Go binary to the statusline install path (the coralline directory under the user's .claude directory), and copy theme files to the install directory when the repository copies differ.

#### Scenario: Successful update deploys new binary

- **WHEN** the user runs the update script and pull, tests, and build all succeed
- **THEN** the installed binary at the statusline install path is replaced with the newly built one and the script exits with code 0

#### Scenario: Test failure blocks deployment

- **WHEN** any test fails during the update script run
- **THEN** the script exits with a non-zero code before building, and the previously installed binary remains unchanged

#### Scenario: Build failure leaves installation intact

- **WHEN** the Go build fails during the update script run
- **THEN** the script exits with a non-zero code and the previously installed binary remains unchanged


<!-- @trace
source: add-update-script
updated: 2026-07-16
code:
  - update.sh
  - README.zh-TW.md
  - update.ps1
  - README.md
-->

---
### Requirement: Update script is idempotent

Running the update script twice in a row SHALL be safe: the second run detects no new commits, still verifies tests and rebuilds, and leaves the installation in the same working state.

#### Scenario: Re-run with no new commits

- **WHEN** the user runs the update script when the branch is already up to date
- **THEN** the script completes successfully without error and the statusline keeps working

<!-- @trace
source: add-update-script
updated: 2026-07-16
code:
  - update.sh
  - README.zh-TW.md
  - update.ps1
  - README.md
-->