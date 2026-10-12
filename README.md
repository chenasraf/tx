# tx

A tmux session manager that creates sessions from YAML configuration files.

![Release](https://img.shields.io/github/v/release/chenasraf/tx)
![Downloads](https://img.shields.io/github/downloads/chenasraf/tx/total)
![License](https://img.shields.io/github/license/chenasraf/tx)

---

## 🚀 Features

- Create tmux sessions with predefined window layouts
- Complex pane splits (horizontal/vertical, nested)
- Run commands in panes on session creation
- Fuzzy finder for session selection
- Config file includes (compose configs from multiple files)
- Quick project session creation from configurable projects directory
- Session/window picker and coding-agent pane picker with live previews of each window's real
  pane layout
- tmux popups that follow the terminal's size

---

## 🎯 Installation

### Download Precompiled Binaries

Precompiled binaries for `tx` are available for **Linux**, **macOS** and **Windows**:

- Visit the [Releases Page](https://github.com/chenasraf/tx/releases/latest) to download the latest
  version for your platform.

### Homebrew (macOS/Linux)

Install from a custom tap:

```bash
brew install chenasraf/tap/tx
```

### Go Install

```bash
go install github.com/chenasraf/tx@latest
```

### Build from Source

```bash
git clone https://github.com/chenasraf/tx.git
cd tx
make build # build only
make install # build & install to ~/.local/bin
```

---

## 🔧 Usage

```bash
# Open a session (fuzzy finder if no name given)
tx [session-name]

# List all configurations and active sessions
tx list
tx ls -b              # bare output (just names)
tx ls -s              # show only active sessions

# Show configuration details
tx show <name>
tx show <name> -j     # JSON output

# Edit configuration file
tx edit
tx edit -c ~/local.yaml   # edit a specific config file

# Create a temporary session
tx create
tx create -r ~/myproject -w src -w lib
tx create -s          # save to config
tx create -S          # save only (don't create)

# Quick project session from projects directory
tx prj [name]
tx prj -s             # save to config

# Create session in background (don't switch to it)
tx -B my-session
tx create -B
tx prj -B myproject

# Add the windows to the current session instead of creating a new one (inside tmux)
tx -a my-session
tx prj -a myproject

# Attach to existing session
tx attach [name]

# Remove configurations
tx rm <name>
tx rm foo bar baz             # remove multiple at once
tx rm <name> -c ~/local.yaml  # remove from a specific config file

# Kill running sessions
tx kill               # kill current session
tx kill <name>        # kill specific session
tx kill foo bar baz   # kill multiple sessions

# Pickers with live previews (inside tmux)
tx sessions           # sessions and windows
tx agents             # panes running a coding agent

# Popups that follow the terminal's size
tx popup open <client> <name> <dir> [display-popup flags...] -- <command...>
```

### Global Flags

| Flag               | Description                                          |
| ------------------ | ---------------------------------------------------- |
| `-V, --verbose`    | Verbose logging                                      |
| `-d, --dry`        | Dry run (show commands without executing)            |
| `-B, --background` | Create session in background without switching to it |

### Pickers

`tx sessions` lists your tmux sessions as a tree you can expand into windows, and `tx agents`
lists the panes running a coding agent ([Claude Code](https://claude.com/claude-code) by default).
Both show a live preview of the highlighted row: the window's real pane layout, scaled to fit,
with each pane's contents in color, or the agent's pane.

Bind them in `tmux.conf` to open in a popup:

```tmux
bind s run-shell "tx sessions --popup --client #{client_name}"
bind a run-shell "tx agents --popup --client #{client_name}"
# reopen popups at the new size when the terminal resizes
set-hook -g client-resized[10] 'run-shell -b "tx popup resized #{q:hook_client}"'
```

| Key           | Sessions                                    | Agents                               |
| ------------- | ------------------------------------------- | ------------------------------------ |
| `↑↓` / `jk`   | Move                                        | Move                                 |
| `→←` / `lh`   | Expand / collapse a session                 |                                      |
| `enter`       | Switch to the session or window             | Switch to the pane                   |
| `0`-`9`       | Switch to the numbered session              | Switch to the numbered pane          |
| `i`           |                                             | Work in the pane without leaving     |
| `/`           | Search                                      | Search                               |
| `esc`         | Leave search, or quit                       | Leave search, or quit                |
| `dd`          | Kill the session or window                  | Kill the agent (SIGTERM, then KILL)  |
| `q`           | Quit                                        | Quit                                 |

While working in an agent's pane with `i`, `ctrl-c` returns to the list and `ctrl-d` closes the
picker. The list follows changes made elsewhere as they happen, and the preview re-captures the
panes every second (`--refresh`).

| Flag          | Description                                                                 |
| ------------- | --------------------------------------------------------------------------- |
| `--popup`     | Open in a popup on `--client` that follows the terminal's size              |
| `--client`    | The client that `enter` switches (default: the one viewing the pane)        |
| `--size`      | Popup size: `0` borderless full screen, `1`-`100` a bordered percentage     |
| `--refresh`   | How often the preview re-captures pane contents (default: `1s`)             |
| `--command`   | `tx agents` only: an agent to list, repeatable (see [Picker](#picker))      |

`--size` falls back to `$TMUX_SWITCHER_SIZE`, then the `picker.size` setting. The pickers need
tmux 3.2 or later.

### Popups

tmux sizes a popup once, when it opens. `tx popup open` runs the command in a hidden session and
holds the popup around a client attached to it, so with the `client-resized` hook above, the popup
is reopened at the new size while the command keeps running undisturbed:

```tmux
bind g run-shell "tx popup open #{q:client_name} lazygit #{q:pane_current_path} -w 90% -h 90% -b rounded -T ' lazygit ' -- lazygit"
```

Flags before `--` go to `display-popup`. A single-word command runs through the shell, as in tmux;
several words run directly.

---

## 📚 Configuration

tx searches for configuration files in these locations (in order):

1. Home directory (`~`)
2. `$XDG_CONFIG_HOME` (if set)
3. `~/.config`
4. `%APPDATA%` (Windows, if set)

File patterns searched:

- `tmux.yaml` / `tmux.yml`
- `.tmux.yaml` / `.tmux.yml`

You can compose your configuration from multiple files using the `include` setting under `.config`.
Included files are merged with the main config, with later includes taking precedence. See
[Config Includes](#config-includes) for details.

### Configuration Format

```yaml
# Simple session
myproject:
  root: ~/Dev/myproject
  windows:
    - ./src
    - ./lib
    - ./test

# Session with named windows
webapp:
  root: ~/Dev/webapp
  windows:
    - name: editor
      cwd: ./src
    - name: server
      cwd: ./backend
      layout:
        cwd: .
        cmd: npm run dev
        split:
          direction: v
          child:
            cwd: .
            cmd: npm run watch

# Session with initial window override
webapp:
  root: ~/Dev/webapp
  initial_window: 0 # open on the "general" window instead of the first configured one
  windows:
    - ./src
    - ./lib

# Session with complex layout
fullstack:
  root: ~/Dev/fullstack
  blank_window: true # add a blank window at the start
  windows:
    - name: dev
      cwd: ./frontend
      layout:
        cwd: .
        cmd: npm start
        split:
          direction: h
          child:
            cwd: ../backend
            cmd: go run .
            split:
              direction: v
              child:
                cwd: .
                cmd: tail -f logs/app.log
```

### Window Configuration

Windows can be specified as:

**String** - just a directory path:

```yaml
windows:
  - ./src
  - ./lib
```

**Object** - with name, cwd, and optional layout:

```yaml
windows:
  - name: mywindow
    cwd: ./src
    layout: ...
```

### Layout Configuration

Layouts define pane splits and commands. When no layout is specified, the following default is used:

```yaml
# Default layout - horizontal split with vertical sub-split and clock
layout:
  cwd: .
  split:
    direction: h
    child:
      cwd: .
      split:
        direction: v
        child:
          cwd: .
          clock: true
```

You can customize the default for all windows via [`default_layout`](#default-layout) in global
settings, or override per window.

#### Layout Formats

**String** - just a directory:

```yaml
layout: ./src
```

**Array** - horizontal splits:

```yaml
layout:
  - ./src
  - ./lib
  - ./test
```

**Object** - full pane configuration:

```yaml
layout:
  cwd: .
  cmd: npm start # command to run
  zoom: true # zoom this pane
  clock: false # show tmux clock mode
  split:
    direction: h # h (horizontal) or v (vertical)
    size: 30 # percentage of space for the child pane (1-100, default: 50)
    child:
      cwd: ./other
      cmd: npm test
      split: # nested splits
        direction: v
        size: 40
        child:
          cwd: .
          clock: true # show clock in this pane
```

#### Pane Options

| Setting | Description                              |
| ------- | ---------------------------------------- |
| `cwd`   | Working directory (defaults to window's) |
| `cmd`   | Command to run in the pane               |
| `zoom`  | Zoom this pane (default: false)          |
| `clock` | Show tmux clock mode (default: false)    |
| `split` | Split this pane (see below)              |

#### Split Options

| Setting     | Description                                                 |
| ----------- | ----------------------------------------------------------- |
| `direction` | `h` (horizontal / side-by-side) or `v` (vertical / stacked) |
| `size`      | Percentage of space for the child pane (1-100, default: 50) |
| `child`     | Pane configuration for the new pane created by the split    |

### Global Settings

The special `.config` key is reserved for global settings and won't be treated as a session:

```yaml
.config:
  shell: /bin/zsh
  projects_path: ~/Dev

myproject:
  root: ~/Dev/myproject
  # ...
```

#### Available Settings

| Setting          | Description                                                          |
| ---------------- | -------------------------------------------------------------------- |
| `shell`          | Shell to use for command execution                                   |
| `projects_path`  | Directory for `tx prj` command (required for prj)                    |
| `default_layout` | Default pane layout for new windows (see below)                      |
| `named_layouts`  | Reusable named layouts (see below)                                   |
| `initial_window` | Window index to select on session creation (default: `1`, see below) |
| `include`        | List of additional config files to merge (see below)                 |
| `picker`         | Settings for `tx sessions` and `tx agents` (see below)               |

#### Default Layout

The `default_layout` setting overrides the built-in default pane arrangement for all windows. It
accepts the same [pane options](#pane-options) as a regular layout.

```yaml
.config:
  default_layout:
    cwd: .
    clock: true
```

#### Named Layouts

Define reusable layouts that can be referenced by name in session configurations:

```yaml
.config:
  named_layouts:
    dev:
      cwd: .
      cmd: npm run dev
      split:
        direction: h
        child:
          cwd: .
          cmd: npm run test:watch
    simple:
      cwd: .
      clock: true

myproject:
  root: ~/Dev/myproject
  windows:
    - name: main
      cwd: .
      layout: dev # references the "dev" named layout
    - name: logs
      cwd: ./logs
      layout: simple # references the "simple" named layout
```

#### Initial Window

The `initial_window` setting controls which window is selected when a new session is created. Window
`0` is the "general" window (created automatically with the session), and configured windows start
at index `1`. The default is `1` (the first configured window).

With `--add`, no "general" window is created: `0` keeps the current window selected, and `N` selects
the `N`th added window.

This can be set globally under `.config` and overridden per session:

```yaml
.config:
  initial_window: 1 # global default

myproject:
  root: ~/Dev/myproject
  initial_window: 0 # override: start on the "general" window
  windows:
    - ./src
    - ./lib
```

#### Config Includes

The `include` setting lets you compose your configuration from multiple files. This is useful for
separating machine-specific configs from shared ones, or simply organizing a large config into
smaller pieces.

```yaml
# ~/.tmux.yaml
.config:
  include:
    - ./local.yaml # relative to this file's directory
    - ~/shared/team.yaml # ~ is expanded to home directory
    - /etc/tx/company.yaml # absolute paths work too

myproject:
  root: ~/Dev/myproject
```

**Path resolution:**

- **Relative paths** are resolved relative to the directory of the config file containing the
  `include`
- **`~`** is expanded to the home directory
- **Absolute paths** are used as-is

Included files can themselves contain `include` lists (nested includes). Circular includes are
detected and skipped. Later includes take precedence over earlier ones when merging.

#### Picker

Settings for [the pickers](#pickers). Flags given on the command line take precedence.

```yaml
.config:
  picker:
    size: 0 # popup size: 0 borderless full screen, 1-100 a bordered percentage
    refresh: 1s # how often the preview re-captures pane contents
    preview_width: # preview width, as a percentage of the popup
      sessions: 78
      agents: 70
    agents:
      commands: [claude, /gemini|codex/]
      match: auto
    colors: # override theme colors by role
      sky: '#89dceb'
```

`agents.commands` lists what counts as an agent. A plain name matches the pane's process name. An
entry in slashes is a regular expression over the pane's foreground command line, which finds
agents started through an interpreter (`node .../gemini`). `agents.match` sets how entries are
checked:

| Value  | Behavior                                                                          |
| ------ | --------------------------------------------------------------------------------- |
| `auto` | Names against the process name, `/regex/` entries against the command line        |
| `name` | Every entry against the process name                                              |
| `args` | Every entry against the command line; a name also matches an interpreter's script |

Colors come from `colors`, then the `@thm_*` options the
[Catppuccin tmux theme](https://github.com/catppuccin/tmux) sets, then a built-in Catppuccin Mocha
palette. The roles are `sky`, `overlay_0`, `overlay_1`, `blue`, `green`, `subtext_0`, `text`,
`peach` and `red`.

#### Shell Resolution Order

The shell used for executing commands is determined in this order:

1. **Config file** - `.config.shell` in your config file (highest priority)
2. **Environment** - `$SHELL` environment variable
3. **Auto-detect** - First available of `/bin/zsh`, `/bin/bash`, `/bin/sh`

---

## 📂 Examples

### Basic Development Setup

```yaml
# ~/.tmux.yaml
dotfiles:
  root: ~/.dotfiles
  windows:
    - .
    - ./utils

webapp:
  root: ~/Dev/webapp
  windows:
    - name: code
      cwd: ./src
    - name: server
      cwd: .
      layout:
        cwd: .
        cmd: npm run dev
        split:
          direction: h
          child:
            cwd: .
            cmd: npm run test:watch
```

### Quick Session

```bash
# Create a session for current directory
tx create

# Create with specific windows
tx create -r ~/myproject -w src -w lib -w test

# Create and save to config
tx create -r ~/myproject -s

# Create in background (don't switch to it)
tx create -B -r ~/myproject
```

### Project Workflow

First, configure your projects directory in `.config`:

```yaml
.config:
  projects_path: ~/Dev
```

Then use `tx prj` to quickly open projects:

```bash
# Select from projects directory with fuzzy finder
tx prj

# Open specific project
tx prj myproject

# Open and save to config for future use
tx prj myproject -s
```

---

## Migrating from v1.x to v2.x

Run `tx migrate` to automatically migrate your configuration. It finds your existing `tmux_local`
config file and adds it as an `include` in your main config. Use `tx migrate -d` for a dry run to
preview changes before applying.

### Local config replaced with includes

In v1.x, tx automatically searched for a `tmux_local.yaml` (or `.tmux_local.yaml`) file and merged
it with the global config. In v2.x, this implicit behavior is replaced with an explicit `include`
mechanism in `.config`.

**Before (v1.x):**

```
~/.tmux.yaml          # global config (auto-discovered)
~/.tmux_local.yaml    # local config (auto-discovered and merged)
```

**After (v2.x):**

```yaml
# ~/.tmux.yaml
.config:
  include:
    - ./tmux_local.yaml # explicitly include the local config
```

Your `tmux_local.yaml` file contents do not need to change — just add the `include` entry to your
main config.

### `--local` flag replaced with `--config`

Commands that accepted `--local` / `-l` (such as `edit`, `create`, `remove`, `prj`) now use
`--config` / `-c` instead, which takes a file path argument.

**Before (v1.x):**

```bash
tx edit -l
tx rm myproject -l
tx create -s -l
tx prj myproject -s -l
```

**After (v2.x):**

```bash
tx edit -c ~/tmux_local.yaml
tx rm myproject -c ~/tmux_local.yaml
tx create -s -c ~/tmux_local.yaml
tx prj myproject -s -c ~/tmux_local.yaml
```

---

## 🛠️ Contributing

I am developing this package on my free time, so any support, whether code, issues, or just stars is
very helpful to sustaining its life. If you are feeling incredibly generous and would like to donate
just a small amount to help sustain this project, I would be very very thankful!

<a href='https://ko-fi.com/casraf' target='_blank'>
  <img height='36' style='border:0px;height:36px;'
    src='https://cdn.ko-fi.com/cdn/kofi1.png?v=3'
    alt='Buy Me a Coffee at ko-fi.com' />
</a>

I welcome any issues or pull requests on GitHub. If you find a bug, or would like a new feature,
don't hesitate to open an appropriate issue and I will do my best to reply promptly.

---

## 📜 License

MIT
