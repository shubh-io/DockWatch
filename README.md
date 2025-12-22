
<h1 align="center">DockMate 🐳</h1>
<p align="center"><b>A terminal-based Docker container manager that actually works.</b></p>

<p align="center">
  <span><img src="https://wakatime.com/badge/github/shubh-io/DockMate.svg" /></span>
  <span><img src="https://img.shields.io/github/stars/shubh-io/DockMate?style=flat&logo=github" /></span>
  <span><img src="https://img.shields.io/github/v/release/shubh-io/DockMate?color=green" /></span>
  <span><img src="https://img.shields.io/github/license/shubh-io/DockMate" /></span>
  <span><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" /></span>
  <span><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-blue?logo=go&logoColor=white" /></span>
  <span><img src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS-blue?style=flat&logo=linux&logoColor=white" /></span>
</p>

> **Note**: Previously named **DockWatch** (renamed to avoid confusion with another project).

![DockMate demo gif](assets/demo.gif)

---

## Overview

DockMate is a **TUI (Text User Interface)** for **managing Docker containers** directly from your terminal.  
Think of `htop`, but for Docker.

- See live container stats at a glance
- Start, stop, restart, and remove containers with single keypresses
- Jump into logs or an interactive shell instantly
- Change Runtime (switch Docker ⇄ Podman)
---

## Comparison
<div align="center">

### DockMate vs LazyDocker

| Feature | DockMate | LazyDocker |
|---------|----------|------------|
| **Installation** | One-command + Homebrew | Homebrew + Multiple package managers |
| **Auto-update** | ✅ Built-in (`dockmate update`) | ❌ Manual updates required |
| **Container loading** | ✅ **Fast (< 2 seconds)** | Slower (variable) |
| **UI Framework** | ✅ **Bubble Tea (modern)** | gocui (older library) |
| **Dependencies** | ✅ **Minimal** (bash, curl) | Multiple system dependencies |
| **Resource usage** | ✅ **Lightweight** | Heavier footprint |
| **Container stats** | ✅ Real-time (CPU, memory, network, disk I/O) | Real-time + ASCII graphs |
| **Docker Compose** | ✅ Full support | ✅ Full support |
| **Interactive logs** | ✅ | ✅ |
| **Shell access** | ✅ One keypress | ✅ |
| **Multi-runtime support** | ✅ **Docker + Podman (native)** | Docker only (Podman via workaround) |
| **Runtime switching** | ✅ **In TUI settings** | ❌ Restart + change env vars |
| **Podman Compose** | ✅ **Auto-detected** | ⚠️ Manual configuration |
| **Image management** | ⏳ Planned | ✅ Layer inspection & pruning |
| **Metrics graphs** | ❌ Text-based (lighter) | ✅ Customizable ASCII graphs |
| **Mouse support** | ❌ Keyboard-focused | ✅ |
| **Best for** | Speed, simplicity, **+ Podman support** | Feature-rich Docker power users |

</div>

### Choose DockMate if you:
- ⚡ Want a **fast, lightweight** Docker TUI
- ⌨️ Prefer **keyboard-driven** workflows
- 📦 Value **simplicity** and **auto-updates**
- 🔄 **Bonus:** Need Podman support (native, zero config)

### Choose LazyDocker if you:
- 📊 Need **ASCII graphs** and visualizations
- 🔍 Want **image layer inspection**
- 🖱️ Prefer **mouse support**
- 🏆 Want a **mature, battle-tested** tool

**Both are excellent - DockMate for speed & simplicity, LazyDocker for advanced features!** 🐳

---

## Features

### 🐳 Docker Management
- Docker and Docker Compose support
- Live metrics (CPU, memory, network I/O, disk I/O)
- Start/stop/restart with one keypress
- Real-time log streaming
- Interactive shell access
- Sort by any column

### ⚡ Performance & UX
- Fast startup 
- Lightweight 
- Fully keyboard-driven
- Persistent settings (`~/.config/dockmate/config.yml`)
- Configurable auto-refresh
- Clean terminal resizing

### 🚀 Bonus: Multi-Runtime Support
- Native Podman support
- Runtime switching (Docker ⇄ Podman)
- Supports Podman Compose
- Helpful error guidance for Podman setup

## Requirements

### Runtime
- **Docker** (recommended) or **Podman** installed and running

### Operating System
- **Linux** (Ubuntu, Debian, Fedora, Arch, etc.)
- **macOS**

### Building from Source (optional)
- **Go 1.24+** required

---

## System Dependencies

DockMate uses the following system tools:

- **curl** - For one-command installation

**macOS:** systemctl checks are automatically skipped.


---

## Installation

### 🍺 Homebrew (Recommended)

```
brew install shubh-io/tap/dockmate
```

Works on both **Linux** and **macOS**. Easiest way to install and update.

### 📦 Quick Install Script

```
curl -fsSL https://raw.githubusercontent.com/shubh-io/DockMate/main/install.sh | sh
```

If that ever fails on your setup, use the two-step variant:

```
curl -fsSL https://raw.githubusercontent.com/shubh-io/DockMate/main/install.sh -o install.sh
sh install.sh
```

### Alternative: User-local installation

If you encounter permission issues with `/usr/local/bin`, install to your user directory instead:

```
curl -fsSL https://raw.githubusercontent.com/shubh-io/dockmate/main/install.sh | INSTALL_DIR=$HOME/.local/bin sh
```

Then add to your PATH. Choose based on your shell:

**For Bash** (most Linux):
```
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

**For Zsh** (macOS default):
```
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

**Not sure which shell?** Run `echo $SHELL` to check.



---

**Note:** Some shells cache executable locations. If `dockmate` isn't found immediately after
installation, refresh your shell's command cache with:

```
hash -r
```

Or open a new terminal session.

### 🔨 Build from Source

If you want to tweak or contribute:

```
git clone https://github.com/shubh-io/DockMate
cd DockMate
go build -o dockmate

# Run locally
./dockmate

# Optional: make it available system-wide
sudo mv dockmate /usr/local/bin/
```

### 🔄 Updating

**Homebrew:**
```
brew upgrade shubh-io/tap/dockmate
```

**Built-in updater:**
```
dockmate update
```

**Or re-run the installer:**
```
curl -fsSL https://raw.githubusercontent.com/shubh-io/DockMate/main/install.sh | sh
```

---

## Usage

```
dockmate
```

Use the keyboard to navigate and control containers.

**Check installed version:**
```
dockmate version
# or
dockmate -v
# or
dockmate --version
```

---

## Keyboard shortcuts

| Key               | Action                          |
|-------------------|---------------------------------|
| `↑ / ↓` or `j / k`| Navigate containers             |
| `Tab`             | Switch to column selection mode |
| `← / →` or `h / l`| Move between columns            |
| `Enter`           | Sort by selected column         |
| `s`               | Start container                 |
| `x`               | Stop container                  |
| `r`               | Restart container               |
| `l`               | View logs                       |
| `e`               | Open interactive shell          |
| `c`               | Compose view                    |
| `d`               | Remove container                |
| `q` or `Ctrl+C`   | Quit                            |

---

## Changing runtime 🛠️

You can switch DockMate's container runtime (Docker ⇄ Podman) in two ways:

- In the TUI: open the Settings panel, change the **Runtime** option to `docker` or `podman`, then save - the new value is persisted to your config and applied after the app restarts.
- From the command line: run the interactive runtime selector:

```
dockmate --runtime
```

This will show a list selector (Docker / Podman) that saves your choice to `~/.config/dockmate/config.yml` (or `$XDG_CONFIG_HOME/dockmate/config.yml`), and you can then restart DockMate normally.


## Verifying downloads (optional)

Release binaries are published with matching SHA256 checksum files.

Example for verifying a release:

```
# Download binary and checksum
curl -fsSL -o dockmate-linux-amd64 \
  https://github.com/shubh-io/DockMate/releases/download/v0.0.8/dockmate-linux-amd64

curl -fsSL -o dockmate-linux-amd64.sha256 \
  https://github.com/shubh-io/DockMate/releases/download/v0.0.8/dockmate-linux-amd64.sha256

# Verify on Linux
sha256sum -c dockmate-linux-amd64.sha256

# Or on macOS
shasum -a 256 -c dockmate-linux-amd64.sha256
```

The installer script will also try to fetch and verify the corresponding `.sha256` file automatically.  
If verification fails, installation is aborted.

---

## Why DockMate?

Most Docker TUIs either try to do too much or require config and setup.  
DockMate aims to be:

- Lightweight
- Zero-config
- "Install and go" for daily container management work

---

## Roadmap

- [x] Docker Compose integration  
- [ ] Container search / filter  
- [ ] Resource monitoring alerts & notifications
- [ ] Image management
- [x] Homebrew distribution
- [x] macOS support

Have ideas? Open an issue.

---

## Contributing

Bug reports, feature requests, and pull requests are all welcome.

1. Fork the repo
2. Create a feature branch
3. Open a PR with a clear description

---

## License

MIT License – do pretty much whatever you want, just keep the license intact.

---

## Credits

Built by [@shubh-io](https://github.com/shubh-io) while learning Go and Docker.  
If DockMate saves you some keystrokes, consider dropping a ⭐ on the repo.

