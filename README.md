
<h1 align="center">DockMate 🐳</h1>
<p align="center"><b>A terminal-based Docker container manager that actually works.</b></p>

<p align="center">
  <span><img src="https://wakatime.com/badge/github/shubh-io/DockMate.svg" /></span>
  <span><img src="https://img.shields.io/github/stars/shubh-io/DockMate?style=flat&logo=github" /></span>
  <span><img src="https://img.shields.io/github/v/release/shubh-io/DockMate?color=green" /></span>
  <span><img src="https://img.shields.io/github/license/shubh-io/DockMate" /></span>
  <span><img src="https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go&logoColor=white" /></span>
  <span><img src="https://img.shields.io/badge/TUI-Bubble%20Tea-blue?logo=go&logoColor=white" /></span>
  <span><img src="https://img.shields.io/badge/Platform-Linux-blue?style=flat&logo=linux&logoColor=white" /></span>

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
- Linux (primarily tested on Debian/Ubuntu)
- Go 1.24+ **only if** building from source

> Should also work on macOS with Docker, Windows is currently untested.

---

## Installation

You can install DockMate with a one-liner, or build from source if you prefer.

### Quick Install (recommended)

```
curl -fsSL https://raw.githubusercontent.com/shubh-io/dockmate/main/install.sh | bash
```

If that ever fails on your setup, use the two-step variant:

```
curl -fsSL https://raw.githubusercontent.com/shubh-io/dockmate/main/install.sh -o install.sh
bash install.sh
```

### Build from source

If you want to tweak or contribute:

```
git clone https://github.com/shubh-io/dockmate
cd dockmate
go build -o dockmate

# Run locally
./dockmate

# Optional: make it available system-wide
sudo mv dockmate /usr/local/bin/
```

### Updating

```
# Built-in updater
dockmate update

# Or simply re-run the installer
curl -fsSL https://raw.githubusercontent.com/shubh-io/dockmate/main/install.sh | bash
```

---

## Verifying downloads (optional)

Release binaries are published with matching SHA256 checksum files.

Example for `v0.0.2`:

```
# Download binary and checksum
curl -fsSL -o dockmate-linux-amd64 \
  https://github.com/shubh-io/dockmate/releases/download/v0.0.2/dockmate-linux-amd64

curl -fsSL -o dockmate-linux-amd64.sha256 \
  https://github.com/shubh-io/dockmate/releases/download/v0.0.2/dockmate-linux-amd64.sha256

# Verify on Linux
sha256sum -c dockmate-linux-amd64.sha256

# Or on macOS
shasum -a 256 -c dockmate-linux-amd64.sha256
```

The installer script will also try to fetch and verify the corresponding `.sha256` file automatically.  
If verification fails, installation is aborted.

---

## Usage

```
dockmate
```

Use the keyboard to navigate and control containers.

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

## Why DockMate?

Most Docker TUIs either try to do too much or require config and setup.  
DockMate aims to be:

- Lightweight
- Zero-config
- “Install and go” for day-to-day container work

---

## Roadmap

- [ ] Docker Compose integration  
- [ ] Container search / filter  
- [ ] `.deb` package

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
