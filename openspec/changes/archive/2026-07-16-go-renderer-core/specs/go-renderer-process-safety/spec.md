## ADDED Requirements

### Requirement: Hard process deadline (watchdog)

The renderer process SHALL arm an in-process watchdog at startup that terminates the process with exit code 1 and no stdout output if rendering has not completed within 5 seconds. The watchdog MUST fire regardless of where the main logic is blocked, including blocking syscalls such as stdin reads, file I/O, and child-process waits.

#### Scenario: Stdin that never delivers EOF cannot create a zombie

- **WHEN** the renderer is started with a stdin pipe whose write end is held open and never closed
- **THEN** the process exits on its own within 5 seconds with exit code 1 and empty stdout

### Requirement: Bounded stdin consumption

The renderer SHALL read at most 4 MiB from stdin. Input beyond that limit SHALL be ignored, and parsing SHALL proceed on the truncated data (degrading to empty fields if the truncation breaks the JSON).

#### Scenario: Oversized input does not grow memory unboundedly

- **WHEN** more than 4 MiB of data is piped to stdin
- **THEN** the renderer stops reading at 4 MiB and completes rendering without error output

### Requirement: Zero MSYS-family child processes

A render SHALL NOT spawn any MSYS/Cygwin-family process (bash, awk, coreutils, or any binary linked against the MSYS runtime). The only permitted child process is the native `git.exe`. All data transformation previously delegated to jq, awk, date, ls, grep, sort, and stty SHALL be performed in-process.

#### Scenario: Process tree during render

- **WHEN** a render executes with all eight core segments enabled
- **THEN** the renderer's child-process tree contains at most one git.exe and no other processes

### Requirement: Git child process is time-bounded and lock-free

The renderer SHALL invoke `git -C <cwd> status --porcelain=v2 --branch` with the environment variable GIT_OPTIONAL_LOCKS=0 and a 2.5-second timeout. On timeout the child process MUST be killed (no orphan git.exe) and the git segment SHALL be hidden. On any git failure the git segment SHALL be hidden without error output.

#### Scenario: Hung git does not hang the render

- **WHEN** the git invocation exceeds 2.5 seconds
- **THEN** the git child is killed, the git segment is absent from the output, and the render completes within the watchdog deadline
