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
- Global and local config file support (with merging)
- Quick project session creation from configurable projects directory

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
tx edit -l            # edit local config

# Create a temporary session
tx create
tx create -r ~/myproject -w src -w lib
tx create -s          # save to config
tx create -S          # save only (don't create)

# Quick project session from projects directory
tx prj [name]
tx prj -s             # save to config

# Attach to existing session
tx attach [name]

# Remove a configuration
tx rm <name>
tx rm <name> -l       # remove from local config

# Kill a running session
tx kill               # kill current session
tx kill <name>        # kill specific session
```

### Global Flags

| Flag            | Description                               |
| --------------- | ----------------------------------------- |
| `-v, --verbose` | Verbose logging                           |
| `-d, --dry`     | Dry run (show commands without executing) |

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

Local config files (`.tmux_local.yaml`) are merged with global config, with local values taking
precedence.

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

Layouts define pane splits and commands:

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
    child:
      cwd: ./other
      cmd: npm test
      split: # nested splits
        direction: v
        child:
          cwd: .
          clock: true # show clock in this pane
```

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

| Setting          | Description                                       |
| ---------------- | ------------------------------------------------- |
| `shell`          | Shell to use for command execution                |
| `projects_path`  | Directory for `tx prj` command (required for prj) |
| `default_layout` | Default pane layout for new windows (see below)   |

#### Default Layout

The `default_layout` setting configures the default pane arrangement for windows. Each pane can
have:

| Setting | Description                                                   |
| ------- | ------------------------------------------------------------- |
| `cwd`   | Working directory (defaults to window's directory)            |
| `cmd`   | Command to run (defaults to none)                             |
| `clock` | Show tmux clock mode (defaults to false)                      |
| `split` | Create a split with direction (`h` or `v`) and a `child` pane |

Example - single pane with clock:

```yaml
.config:
  default_layout:
    cwd: .
    clock: true
```

Example - horizontal split with vertical sub-split (default):

```yaml
.config:
  default_layout:
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
