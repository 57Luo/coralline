## ADDED Requirements

### Requirement: Path collapsing supports Windows separators

The dir segment's path collapsing SHALL work for both POSIX (`/`) and Windows (`\`) path separators. When the input path contains a backslash, the renderer SHALL split on and rejoin with backslashes; otherwise it SHALL use forward slashes. Behavior for POSIX paths SHALL remain byte-for-byte identical to the current implementation.

#### Scenario: Deep Windows path is collapsed

- **WHEN** workspace.current_dir is a Windows path whose component count exceeds the configured depth
- **THEN** the dir segment shows the first, second, ellipsis, and last components joined by backslashes

##### Example: depth 4 collapse

| Input path | VL_PATH_DEPTH | Output |
| ---------- | ------------- | ------ |
| C:\Users\demo\projects\deep\nested\coralline | 4 | C:\Users\…\coralline |
| ~\projects\deep\nested\coralline | 4 | ~\projects\…\coralline |
| /Users/demo/coralline | 4 | /Users/demo/coralline |
| /Users/demo/projects/deep/nested/coralline | 4 | /Users/…/coralline |

#### Scenario: POSIX behavior unchanged

- **WHEN** workspace.current_dir is a POSIX path
- **THEN** collapsing output is identical to the pre-change implementation and the existing golden render test passes unmodified

