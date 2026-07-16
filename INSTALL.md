# coralline — Installation Guide

> This is the **57Luo/coralline** fork. The statusline hot path has been rewritten
> from bash to a native Go executable to eliminate the MSYS zombie-process problem
> on Windows ([background](./openspec/changes/go-renderer-core/proposal.md)).
> The Go renderer is the primary install path documented below; the original
> bash renderer is retained in the [Appendix](#appendix-bash-renderer-installsh).

## Go renderer (primary path)

### Prerequisites

- **Go toolchain** (1.22+): `go version`
- **Git** (for the `git` segment): `git --version`
- A [Nerd Font](https://www.nerdfonts.com/) terminal, or set `VL_ASCII=1` for glyph-free rendering

No `jq` or bash dependency — the Go renderer is a single native binary.

### 1. Build

From the repo root:

```bash
cd cmd/coralline
go build -o coralline.exe .
```

(On macOS/Linux, omit `.exe`: `go build -o coralline .`)

### 2. Place the binary

Copy the compiled binary to your coralline install directory:

```bash
mkdir -p ~/.claude/coralline
cp coralline.exe ~/.claude/coralline/coralline.exe
```

On Windows, a typical path is `C:/Users/<you>/.claude/coralline/coralline.exe`.

### 3. Register in Claude Code

Add (or update) the `statusLine` entry in `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "C:/Users/<you>/.claude/coralline/coralline.exe",
    "refreshInterval": 10
  }
}
```

On macOS/Linux:

```json
{
  "statusLine": {
    "type": "command",
    "command": "~/.claude/coralline/coralline",
    "refreshInterval": 10
  }
}
```

### 4. Verify

Feed sample input to the binary and confirm it produces output:

```bash
~/.claude/coralline/coralline.exe < example/sample-input.json
```

Success means: exit code `0`, a rendered statusline on stdout, no error on stderr.

### Configuration

The Go renderer reads the same `~/.claude/coralline.conf` and `themes/*.conf`
files as the bash version — no configuration migration needed. See the
[README Configuration section](./README.md#configuration) for all available
variables.

### Current coverage

The Go renderer supports **all 18 segments** (`dir`, `git`, `model`, `effort`,
`ctx`, `limit5h`, `limit7d`, `burn`, `cost`, `clock`, `lines`, `tokens`,
`style`, `duration`, `stash`, `project`, `node`, `python`), plus the **pill**
style and **fixed multi-line** layout (`VL_SEGMENTS` / `VL_SEGMENTS2` /
`VL_SEGMENTS3` / `VL_MAX_LINES`).

**Not yet ported** (available only via the bash renderer): the `lean` /
`classic` styles and the `auto` responsive layout. If you need those, use the
bash renderer instead — see the [Appendix](#appendix-bash-renderer-installsh).

---

## AI Installation Playbook

> **You are an AI coding assistant** and a user asked you to install coralline.
> For this fork, the primary path is building the Go renderer (above). If the
> user's environment has Go installed, follow the Go path. If Go is unavailable,
> fall back to the bash renderer in the Appendix.

When installing for a user:

1. Check whether Go is available (`go version`).
2. If yes: clone the repo, build `cmd/coralline`, place the binary, register in
   `settings.json`, write `~/.claude/coralline.conf`, and verify.
3. If no: fall back to the bash renderer (Appendix), using `install.sh --install-only`.
4. Write `~/.claude/coralline.conf` based on the user's preferences
   (see [AI Interview](#ai-interview)).
5. Verify with sample input.
6. Remind the user to restart Claude Code or open a new session.

### AI Interview

Ask concise questions. If the user says "you decide", choose the defaults.

1. **Theme**: inspect `~/.claude/coralline/themes/**/*.conf` and offer the installed theme
   labels. Default to `claude-coral` when unsure.
2. **Style**: `pill` default. Note: `lean` and `classic` are only available with the bash
   renderer.
3. **Segments**: default is `dir git model ctx limit5h limit7d cost clock`.
   The Go renderer supports all segments.
4. **Layout**: fixed multi-line (`VL_MAX_LINES`). Note: `auto` responsive layout is only
   available with the bash renderer.
5. **Details**: clock format, Nerd Font, project segment, limit sync — same as upstream.

### Write Config

Create `~/.claude/coralline.conf`:

```bash
# coralline config
. "$HOME/.claude/coralline/themes/claude-coral.conf"

VL_STYLE="pill"
VL_LAYOUT="fixed"
VL_MAX_LINES=3
VL_WRAP_MARGIN=4
VL_SEGMENTS="dir git model ctx limit5h limit7d cost clock"
VL_SEGMENTS2=""
VL_SEGMENTS3=""
VL_CLOCK="12h"
VL_CLOCK_SECONDS=1
VL_BAR_WIDTH=5
VL_COST_DECIMALS=2
VL_PATH_DEPTH=4
VL_NAME_MAX=0
VL_ASCII=0
VL_LEAN_SEP=""
```

---

## Appendix: Bash renderer (install.sh)

The original bash renderer supports all 16 segments and all styles (pill, lean,
classic) with responsive auto layout. It requires `jq` and bash at runtime.
Both renderers share the same `coralline.conf` and `themes/*.conf` files — no
configuration changes are needed when switching between them.

### Prerequisites

```bash
command -v jq || echo "MISSING: jq"
command -v curl || echo "MISSING: curl"
```

`jq` is required at runtime. `curl` is only needed for the remote one-line installer.

### Quick install

```bash
curl -fsSL https://raw.githubusercontent.com/57Luo/coralline/main/install.sh | bash
```

Pin a tagged release: `... | bash -s -- --ref v0.9.1`.

Non-interactive (for AI installs):

```bash
curl -fsSL https://raw.githubusercontent.com/57Luo/coralline/main/install.sh | bash -s -- --install-only
```

### Testing a fork

```bash
curl -fsSL https://raw.githubusercontent.com/YOU/coralline/main/install.sh | bash -s -- --repo YOU/coralline
```

### From a local clone

```bash
bash install.sh
```

The installer copies the renderer, wizard, sample input, and themes into
`~/.claude/coralline`, merges the `statusLine` setting with `jq`, and
(interactively) runs the setup wizard.

### Manual install

```bash
git clone https://github.com/57Luo/coralline ~/.claude/coralline-src
mkdir -p ~/.claude/coralline/themes
cp ~/.claude/coralline-src/statusline.sh ~/.claude/coralline/
cp ~/.claude/coralline-src/configure.sh ~/.claude/coralline/
cp ~/.claude/coralline-src/install.sh ~/.claude/coralline/
cp ~/.claude/coralline-src/test/sample-input.json ~/.claude/coralline/sample-input.json
cp ~/.claude/coralline-src/themes/*.conf ~/.claude/coralline/themes/
chmod +x ~/.claude/coralline/statusline.sh ~/.claude/coralline/configure.sh
bash ~/.claude/coralline/configure.sh --install
```

### Register (bash)

Add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "bash ~/.claude/coralline/statusline.sh",
    "refreshInterval": 10
  }
}
```

### Verification (bash)

```bash
CORALLINE_NO_SAMPLE=1 bash ~/.claude/coralline/statusline.sh < ~/.claude/coralline/sample-input.json
```

`CORALLINE_NO_SAMPLE=1` keeps the render read-only so the sample's preview
values don't poison the cross-session limit/burn stores.

### Reconfigure

```bash
bash ~/.claude/coralline/configure.sh
```

Multiple profiles: `bash ~/.claude-personal/coralline/configure.sh --profile=~/.claude-personal`

### Uninstall

```bash
rm -rf ~/.claude/coralline ~/.claude/coralline.conf
```

Then remove the `statusLine` block from `~/.claude/settings.json` (or restore
the newest `settings.json.bak.*`).
