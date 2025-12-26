#!/bin/sh
set -eu

REPO="shubh-io/dockmate"
BINARY_NAME="dockmate"

# ==============================================================================
# Dockmate Installer
# ==============================================================================
# This script will:
#   1. Check for existing Homebrew installation (won't overwrite)
#   2. Detect your system architecture (amd64/arm64)
#   3. Download the latest release from GitHub: https://github.com/shubh-io/dockmate/releases/latest
#   4. Verify checksum if available (for security)
#   5. Install binary to /usr/local/bin (or custom INSTALL_DIR)
#
# Source code: https://github.com/shubh-io/dockmate
# This installer: https://github.com/shubh-io/dockmate/blob/main/install.sh
# ==============================================================================

echo ""
echo ""
echo "====================================================================="
echo "Dockmate Installer 🐳"
echo "====================================================================="
echo ""
echo "This installer will:"
echo "  • Check for existing installations and overwrite if needed"
echo "  • Download the latest release from GitHub"
echo "  • Verify the download with checksums"
echo "  • Install to /usr/local/bin (or \$INSTALL_DIR if set)"
echo ""
echo "Source: https://github.com/$REPO"
echo "Installation script: https://github.com/$REPO/blob/main/install.sh"
echo ""

# Give users time to read the intro
sleep 5

# For Homebrew folks — robust detection (check early to avoid unnecessary work)
# Check via brew metadata first, then path heuristics
if command -v brew >/dev/null 2>&1; then
    # Prefer explicit tap formula; fallback to plain name
    if brew list --versions shubh-io/tap/dockmate >/dev/null 2>&1 || brew list --versions dockmate >/dev/null 2>&1; then
        echo "⚠️  Detected: dockmate is installed via Homebrew"
        echo ""
        echo "To update, please use:"
        echo "  brew upgrade shubh-io/tap/dockmate"
        echo ""
        echo "If you want to switch to script-based installation:"
        echo "  1. brew uninstall dockmate"
        echo "  2. Re-run this installer"
        exit 0
    else
        # Fallback: compare executable path against Homebrew prefix
        if command -v dockmate >/dev/null 2>&1; then
            DOCKMATE_PATH=$(command -v dockmate)
            BREW_PREFIX=$(brew --prefix 2>/dev/null || true)
            if [ -n "$BREW_PREFIX" ]; then
                # Common brew locations to match (Intel/macOS, Apple Silicon, Linuxbrew)
                case "$DOCKMATE_PATH" in
                    "$BREW_PREFIX"*|*"/Cellar/dockmate"*|*"/opt/homebrew"*|*"/usr/local/Cellar"*|*".linuxbrew"*|*"/home/linuxbrew"*)
                        echo "⚠️  Detected: dockmate appears to be installed under Homebrew prefix ($BREW_PREFIX)"
                        echo ""
                        echo "To update, please use:"
                        echo "  brew upgrade shubh-io/tap/dockmate"
                        echo ""
                        echo "If you want to switch to script-based installation:"
                        echo "  1. brew uninstall dockmate"
                        echo "  2. Re-run this installer"
                        exit 0
                        ;;
                esac
            fi
        fi
    fi
fi

# Detect OS and architecture
OS_NAME=$(uname -s)
case "$OS_NAME" in
    Darwin) RELEASE_OS="darwin" ;;
    Linux) RELEASE_OS="linux" ;;
    *) echo "Unsupported OS: $OS_NAME"; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
    x86_64) RELEASE_ARCH="amd64" ;;
    aarch64|arm64) RELEASE_ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

ASSET_NAME="${BINARY_NAME}-${RELEASE_OS}-${RELEASE_ARCH}"

echo "==> Preparing to install dockmate from GitHub releases..."
echo "==> System: $OS_NAME / Architecture: $ARCH ($RELEASE_ARCH)"
echo ""

# Installation directory (default or from environment)

# To change install dir, run:
#   export INSTALL_DIR=$HOME/.local/bin
# or set it inline:
#   INSTALL_DIR=$HOME/.local/bin sh install.sh
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
echo "Installation directory: $INSTALL_DIR"
echo ""

# Check if dockmate already exists on PATH. If found, install into the same directory
# instead of prompting the user (keeps behavior simple and avoids accidental multi-installs).
EXISTING_PATH="$(command -v dockmate 2>/dev/null || true)"
if [ -n "$EXISTING_PATH" ]; then
    if [ "$EXISTING_PATH" != "$INSTALL_DIR/$BINARY_NAME" ]; then
        EXISTING_DIR="$(dirname "$EXISTING_PATH")"
        echo "⚠️  Found existing 'dockmate' executable at: $EXISTING_PATH"
        echo "Installing into existing location: $EXISTING_DIR/"
        INSTALL_DIR="$EXISTING_DIR"
    else
        echo "dockmate already installed at $INSTALL_DIR; installer will overwrite the existing file." 
    fi
fi
# Determine whether sudo is needed
# 1) If running as root (id -u == 0) -> no sudo
# 2) If INSTALL_DIR exists and is writable by current user -> no sudo
# 3) If parent directory is writable (we can create INSTALL_DIR) -> no sudo
# Otherwise -> sudo is required
UID_VALUE=$(id -u 2>/dev/null || echo "")
if [ "$UID_VALUE" = "0" ]; then
    USE_SUDO=0
else
    if [ -d "$INSTALL_DIR" ]; then
        if [ -w "$INSTALL_DIR" ]; then
            USE_SUDO=0
        else
            USE_SUDO=1
        fi
    else
        # Directory doesn't exist - check if we can create it
        PARENT_DIR=$(dirname "$INSTALL_DIR")
        if [ -w "$PARENT_DIR" ]; then
            USE_SUDO=0
        else
            USE_SUDO=1
        fi
    fi
fi

# Check if sudo is available when actually needed
if [ "$USE_SUDO" -eq 1 ]; then
    echo "ℹ️  Note: $INSTALL_DIR requires elevated privileges"
    echo "   This installer will use 'sudo' to:"
    echo "     - Create the directory (if needed)"
    echo "     - Copy the binary to $INSTALL_DIR"
    echo "     - Set executable permissions (chmod 755)"
    echo ""
    if ! command -v sudo >/dev/null 2>&1; then
        echo "Error: sudo is not available on this system"
        echo ""
        echo "Options:"
        echo "  1. Run this script as root"
        echo "  2. Set a writable INSTALL_DIR: export INSTALL_DIR=\$HOME/.local/bin"
        echo ""
        exit 1
    fi
    echo "You may be prompted for your password..."
    echo ""
fi

# Create directory if needed
if [ ! -d "$INSTALL_DIR" ]; then
    if [ "$USE_SUDO" -eq 1 ]; then
        echo "==> Creating directory: $INSTALL_DIR (requires sudo)"
        sudo mkdir -p "$INSTALL_DIR" || {
            echo "Error: Failed to create $INSTALL_DIR"
            exit 1
        }
    else
        echo "==> Creating directory: $INSTALL_DIR"
        mkdir -p "$INSTALL_DIR" || {
            echo "Error: Failed to create $INSTALL_DIR"
            exit 1
        }
    fi
fi

# Better JSON parsing - fetch entire response first
API_URL="https://api.github.com/repos/$REPO/releases/latest"
echo "==> Checking GitHub for the latest release..."

# Detect available download tool
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_TOOL="curl"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_TOOL="wget"
elif command -v fetch >/dev/null 2>&1; then
    DOWNLOAD_TOOL="fetch"
else
    echo "Error: No download tool found (curl, wget, or fetch)"
    echo "Please install curl, wget, or fetch and re-run this installer"
    exit 1
fi

# Download the full API response to avoid pipe issues
case "$DOWNLOAD_TOOL" in
    curl)
        API_RESPONSE=$(curl -fsSL "$API_URL" 2>&1) || {
            echo "Error: Failed to fetch release info from GitHub"
            echo "This might be due to rate limiting or network issues"
            exit 1
        }
        ;;
    wget)
        API_RESPONSE=$(wget -qO- "$API_URL" 2>&1) || {
            echo "Error: Failed to fetch release info from GitHub"
            echo "This might be due to rate limiting or network issues"
            exit 1
        }
        ;;
    fetch)
        API_RESPONSE=$(fetch -qo- "$API_URL" 2>&1) || {
            echo "Error: Failed to fetch release info from GitHub"
            echo "This might be due to rate limiting or network issues"
            exit 1
        }
        ;;
esac

# Parse tag name more reliably
LATEST_TAG=$(echo "$API_RESPONSE" | grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/.*"\([^"]*\)".*/\1/')

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Could not determine latest release version"
    echo "GitHub API might be rate limited. Try again in a few minutes."
    exit 1
fi

echo "✔ Latest version found: $LATEST_TAG"
echo ""
ASSET_NAME="dockmate-linux-${RELEASE_ARCH}"
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/$ASSET_NAME"

echo "==> Downloading release binary..."
echo "==> From: $DOWNLOAD_URL"

TMP_BIN=$(mktemp /tmp/dockmate.XXXXXX)

# Download with better error handling
case "$DOWNLOAD_TOOL" in
    curl)
        if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_BIN"; then
            echo "Error: Failed to download binary"
            rm -f "$TMP_BIN"
            exit 1
        fi
        ;;
    wget)
        if ! wget -qO "$TMP_BIN" "$DOWNLOAD_URL"; then
            echo "Error: Failed to download binary"
            rm -f "$TMP_BIN"
            exit 1
        fi
        ;;
    fetch)
        if ! fetch -qo "$TMP_BIN" "$DOWNLOAD_URL"; then
            echo "Error: Failed to download binary"
            rm -f "$TMP_BIN"
            exit 1
        fi
        ;;
esac

# Checksum verification (optional) — use checksums.txt from the release
CHECKSUMS_URL="https://github.com/$REPO/releases/download/$LATEST_TAG/checksums.txt"
CHECKSUM_DOWNLOADED=0

# Try to download checksums.txt to a temp file next to TMP_BIN
CHECKSUM_FILE_TMP="${TMP_BIN}.checksums"
case "$DOWNLOAD_TOOL" in
    curl)
        if curl -fsSL -o "$CHECKSUM_FILE_TMP" "$CHECKSUMS_URL" 2>/dev/null; then
            CHECKSUM_FILE="$CHECKSUM_FILE_TMP"
        fi
        ;;
    wget)
        if wget -qO "$CHECKSUM_FILE_TMP" "$CHECKSUMS_URL" 2>/dev/null; then
            CHECKSUM_FILE="$CHECKSUM_FILE_TMP"
        fi
        ;;
    fetch)
        if fetch -qo "$CHECKSUM_FILE_TMP" "$CHECKSUMS_URL" 2>/dev/null; then
            CHECKSUM_FILE="$CHECKSUM_FILE_TMP"
        fi
        ;;
esac

if [ -n "${CHECKSUM_FILE:-}" ] && [ -f "$CHECKSUM_FILE" ]; then
    # File format expected: <sha256sum><space><space><filename> or <sha256sum><space><filename>
    CHECKSUM=$(awk -v name="$ASSET_NAME" '$2==name {print $1; exit}' "$CHECKSUM_FILE" 2>/dev/null || true)
    if [ -n "$CHECKSUM" ]; then
        echo "==> Verifying checksum for $ASSET_NAME..."
        VERIFY_FILE=$(mktemp /tmp/dockmate-check.XXXXXX)
        printf '%s  %s\n' "$CHECKSUM" "$TMP_BIN" > "$VERIFY_FILE"

        if command -v sha256sum >/dev/null 2>&1; then
            if sha256sum -c "$VERIFY_FILE" >/dev/null 2>&1; then
                echo "✔ Checksum verified"
                CHECKSUM_DOWNLOADED=1
            else
                echo "Warning: Checksum mismatch for $ASSET_NAME"
            fi
        elif command -v shasum >/dev/null 2>&1; then
            if shasum -a 256 -c "$VERIFY_FILE" >/dev/null 2>&1; then
                echo "✔ Checksum verified (shasum)"
                CHECKSUM_DOWNLOADED=1
            else
                echo "Warning: Checksum mismatch for $ASSET_NAME (shasum)"
            fi
        else
            echo "Warning: no checksum verifier available (install sha256sum or shasum)"
        fi

        rm -f "$VERIFY_FILE"
    else
        echo "==> No checksum entry for $ASSET_NAME in checksums.txt; skipping verification."
    fi

    rm -f "$CHECKSUM_FILE"
else
    echo "==> No checksums.txt found for this release; skipping verification."
fi
echo ""
chmod +x "$TMP_BIN"

# Check if install directory is in PATH
PATH_CHECK=0
IFS=:
for dir in $PATH; do
    if [ "$dir" = "$INSTALL_DIR" ]; then
        PATH_CHECK=1
        break
    fi
done
unset IFS

echo "==> Installing $BINARY_NAME to $INSTALL_DIR..."

# Use sudo only if needed. Try to stop services/processes if target is busy.
install_target="$INSTALL_DIR/$BINARY_NAME"
do_install() {
    src="$1"
    dst="$2"
    if [ "$USE_SUDO" -eq 1 ]; then
        sudo cp "$src" "$dst"
        sudo chmod 755 "$dst" || true
    else
        cp "$src" "$dst"
        chmod 755 "$dst" || true
    fi
}

try_install() {
    do_install "$TMP_BIN" "$install_target" 2>/tmp/dockmate-install.err
    return $?
}

# First attempt
if try_install; then
    rm -f "$TMP_BIN"
else
    # If install failed, attempt to stop systemd service and kill running processes, then retry
    echo "Install failed; attempting to stop running dockmate processes and retry..."
    # kill any running processes by name
    sudo pkill -f dockmate >/dev/null 2>&1 || true
    sleep 1

    if try_install; then
        rm -f "$TMP_BIN"
    else
        # final fallback: save the downloaded binary as .new and instruct user to reboot or manually replace
        NEW_TARGET="$INSTALL_DIR/${BINARY_NAME}.new"
        echo "Could not overwrite $install_target (file busy). Saving new binary as: $NEW_TARGET"
        if [ "$USE_SUDO" -eq 1 ]; then
            sudo mv "$TMP_BIN" "$NEW_TARGET" || {
                echo "Error: Failed to move new binary to $NEW_TARGET"
                rm -f "$TMP_BIN"
                exit 1
            }
            sudo chmod 755 "$NEW_TARGET" || true
        else
            mv "$TMP_BIN" "$NEW_TARGET" || {
                echo "Error: Failed to move new binary to $NEW_TARGET"
                rm -f "$TMP_BIN"
                exit 1
            }
            chmod 755 "$NEW_TARGET" || true
        fi
        echo "The installer was unable to overwrite the running executable due to a 'Text file busy' error."
        echo "A copy of the new binary was saved as: $NEW_TARGET"
        echo "To complete the update, either reboot the machine or replace the file after stopping the process:" 
        echo "  sudo mv $NEW_TARGET $install_target && sudo chmod 755 $install_target"
        exit 0
    fi
fi

echo ""
echo "====================================================================="
echo "✔ Installation Complete!"
echo "====================================================================="
echo ""
echo "Installed:"
echo "  Binary:   $INSTALL_DIR/$BINARY_NAME"
echo "  Version:  $LATEST_TAG"
if [ "${CHECKSUM_DOWNLOADED:-0}" -eq 1 ]; then
    CHECKSUM_STATUS="Verified"
else
    CHECKSUM_STATUS="Not available"
fi

echo "  Checksum: $CHECKSUM_STATUS"
echo ""
echo "NOTE: You can check dockmate version by running:"
echo "  dockmate version"
echo ""
echo "It should show something like:"
echo "  DockMate version: $LATEST_TAG"
echo ""
echo "To run the application now, execute:"
echo "  dockmate"
echo ""
echo "To update later:"
echo "  dockmate update"
echo ""
echo "  or"
# echo "  re-run the installer script manually:" 
echo ""
echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sh"
echo ""

if [ "$PATH_CHECK" -eq 0 ]; then
    echo "⚠️  $INSTALL_DIR is not in your PATH"
    echo ""
    echo "Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.):"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
    echo "Then reload your shell or run: source ~/.bashrc"
    echo ""
fi

echo "If the command isn't found immediately, refresh your shell:"
echo "  hash -r"
echo ""
if [ -n "$EXISTING_PATH" ]; then
    echo "If you previously ran 'dockmate update' and the update reported success but the binary did not change, please re-run the installer directly to force replace the executable:"
    echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | sh"
    echo "or"
    echo "  wget -qO- https://raw.githubusercontent.com/$REPO/main/install.sh | sh"
    echo ""
    # If the user installed to a custom directory, show how to pass INSTALL_DIR
    if [ "$INSTALL_DIR" != "/usr/local/bin" ]; then
        echo "If you installed to a custom directory ($INSTALL_DIR), re-run the installer with INSTALL_DIR set to preserve that location:"
        echo "  curl -fsSL https://raw.githubusercontent.com/$REPO/main/install.sh | INSTALL_DIR=\"$INSTALL_DIR\" sh"
        echo "  wget -qO- https://raw.githubusercontent.com/$REPO/main/install.sh | INSTALL_DIR=\"$INSTALL_DIR\" sh"
        echo ""
    fi
fi

echo "Thank you for using dockmate! 🐳"
echo "====================================================================="
