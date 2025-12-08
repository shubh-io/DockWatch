<div align="center">

# DockMate 🐳
A terminal-based Docker container manager that actually ***works***.

---

<!-- Badges -->
![WakaTime](https://wakatime.com/badge/github/shubh-io/DockMate.svg)
![GitHub Stars](https://img.shields.io/github/stars/shubh-io/DockMate)
![License](https://img.shields.io/github/license/shubh-io/DockMate)
![Pull Requests](https://img.shields.io/github/issues-pr/shubh-io/DockMate)
![Last Commit](https://img.shields.io/github/last-commit/shubh-io/DockMate)
![Repo Size](https://img.shields.io/github/repo-size/shubh-io/DockMate)

</div>

**Note:** Renamed from _DockWatch_ to avoid confusion with existing project


![DockMate demo gif](demo.gif)

## What is this?

Tired of typing `docker ps` a million times? me too. This is a simple TUI (text user interface) that lets you manage docker containers without leaving your terminal. Think htop but for docker.

## Features

- Live container stats (CPU, memory, PIDs)
- Start/stop/restart containers with a single keypress
- View container logs (last 100 lines)
- Interactive shell access
- Sort by any column
- Auto-refreshes every 2 seconds
- Keyboard-driven (no mouse needed)
- Responsive to terminal resize

## Requirements

- Docker installed and running
- Linux (tested on Ubuntu/Debian)
- Go 1.24+ (if building from source)

*Note: Should work on macOS with Docker. Windows support untested.*

## Installation

```
# clone the repo
git clone https://github.com/shubh-io/dockmate
cd dockmate

# build it
go build -o dockmate

# run it
./dockmate

# optional: install globally
sudo mv dockmate /usr/local/bin/
```

## Usage

```
dockmate
```

That's it. Navigate with arrows, press keys to manage containers.

## Keyboard shortcuts

| Key | What it does |
|-----|--------------|
| `↑/↓` or `j/k` | navigate containers |
| `Tab` | switch to column mode |
| `←/→` or `h/l` | navigate columns (in column mode) |
| `Enter` | sort by selected column (in column mode) |
| `s` | start container |
| `x` | stop container |
| `r` | restart container |
| `l` | view logs |
| `e` | open interactive shell |
| `d` | remove container |
| `q` or `Ctrl+C` | quit |

## Why another docker TUI?

Because I wanted something lightweight that just works. No config files, no setup, just run it and manage your containers.

## Roadmap

- [ ] Remote docker host support
- [ ] Resource usage graphs  
- [ ] Docker compose integration
- [ ] Container search/filter
- [ ] .deb package


Got ideas? Open an issue!

## Contributing

Found a bug? Got an idea? Open an issue or send a PR.

## License

MIT License - use it however you want

## Credits

Built by [@shubh-io](https://github.com/shubh-io) while learning Go and Docker.

If you find this useful, star it ⭐
