
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

---

## Comparison
<div align="center">

### DockMate vs LazyDocker


| Feature | DockMate | LazyDocker |
|---------|----------|------------|
| **Installation** | One-command + Homebrew | Homebrew + Multiple package managers |
| **Auto-update** | ✅ Built-in (`dockmate update`) | ❌ Manual updates required |
| **Container loading** | ✅ **Fast (2 seconds)** | Slower (variable) |
| **UI Framework** | ✅ **Bubble Tea (new)** | gocui (older library) |
| **Dependencies** | ✅ **Minimal** (bash, curl) | Multiple system dependencies |
| **Container stats** | ✅ Real-time (CPU, memory, network, disk I/O) | Real-time + ASCII graphs |
| **Interactive logs** | ✅ | ✅ |
| **Shell access** | ✅ One keypress | ✅ |
| **Docker Compose** | ⏳ Planned | ✅ |
| **Image management** | ⏳ Planned | ✅ Layer inspection & pruning |
| **Metrics graphs** | ❌ Text-based (lighter) | ✅ Customizable ASCII graphs |
| **Mouse support** | ❌ Keyboard-focused | ✅ |
| **Resource usage** | ✅ **Lightweight** | Heavier footprint |
| **Best for** | Speed, simplicity, modern workflows | Feature-rich power users |



</div>

### When to use DockMate?

- ✅ You want a modern, lightweight, and fast TUI
- ✅ You prefer keyboard-driven workflows
- ✅ You need quick container monitoring over SSH
- ✅ You want one-command install with auto-updates
- ✅ You value simplicity over features

### When to use LazyDocker?

- ✅ You need Docker Compose management
- ✅ You want metrics graphs and visualizations
- ✅ You need image layer inspection
- ✅ You prefer mouse support
- ✅ You want a mature tool


**Both are great tools - choose based on your workflow!** 🐳


---

## Features

- Live container metrics: CPU, memory, PIDs, network I/O, block I/O
- Start / stop / restart containers
- View recent logs
- Open an interactive shell inside a container
- Sort by any column
- Auto-refresh every 2 seconds
- Fully keyboard-driven (no mouse)
- Resizes cleanly with your terminal

---


## Requirements

- Docker installed and running
- Linux or macOS

- Go 1.24+ **only if** building from source

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
| `d`               | Remove container                |
| `q` or `Ctrl+C`   | Quit                            |

---

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

- [ ] Docker Compose integration  
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

