#!/bin/sh
set -e

REPO="Dilgo-dev/tossit"
BINARY="tossit"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "Unsupported OS: $OS (use install.ps1 for Windows)"; exit 1 ;;
esac

LATEST=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$LATEST" ]; then
    echo "Failed to get latest version"
    exit 1
fi

URL="https://github.com/$REPO/releases/download/$LATEST/$BINARY-$OS-$ARCH"
echo "Downloading tossit $LATEST for $OS/$ARCH..."

INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

curl -fsSL "$URL" -o "$INSTALL_DIR/$BINARY"
chmod +x "$INSTALL_DIR/$BINARY"

if ! echo "$PATH" | grep -q "$INSTALL_DIR"; then
    echo ""
    echo "Add this to your shell profile:"
    echo "  export PATH=\"$INSTALL_DIR:\$PATH\""
    echo ""
fi

# Install shell completions
install_completions() {
    TOSSIT="$INSTALL_DIR/$BINARY"

    # Zsh
    if command -v zsh >/dev/null 2>&1; then
        ZSH_COMP_DIR="${ZDOTDIR:-$HOME}/.zfunc"
        mkdir -p "$ZSH_COMP_DIR"
        "$TOSSIT" completion zsh > "$ZSH_COMP_DIR/_tossit"
        # Ensure .zfunc is in fpath (add to .zshrc if not present)
        ZSHRC="${ZDOTDIR:-$HOME}/.zshrc"
        if [ -f "$ZSHRC" ]; then
            if ! grep -q '\.zfunc' "$ZSHRC" 2>/dev/null; then
                printf '\nfpath=(~/.zfunc $fpath)\nautoload -Uz compinit && compinit\n' >> "$ZSHRC"
            fi
        fi
        echo "Zsh completions installed to $ZSH_COMP_DIR/_tossit"
    fi

    # Bash
    if command -v bash >/dev/null 2>&1; then
        BASH_COMP_DIR=""
        if [ -d "/etc/bash_completion.d" ] && [ -w "/etc/bash_completion.d" ]; then
            BASH_COMP_DIR="/etc/bash_completion.d"
        elif [ -d "$HOME/.local/share/bash-completion/completions" ]; then
            BASH_COMP_DIR="$HOME/.local/share/bash-completion/completions"
        else
            BASH_COMP_DIR="$HOME/.local/share/bash-completion/completions"
            mkdir -p "$BASH_COMP_DIR"
        fi
        "$TOSSIT" completion bash > "$BASH_COMP_DIR/tossit"
        echo "Bash completions installed to $BASH_COMP_DIR/tossit"
    fi

    # Fish
    if command -v fish >/dev/null 2>&1; then
        FISH_COMP_DIR="$HOME/.config/fish/completions"
        mkdir -p "$FISH_COMP_DIR"
        "$TOSSIT" completion fish > "$FISH_COMP_DIR/tossit.fish"
        echo "Fish completions installed to $FISH_COMP_DIR/tossit.fish"
    fi
}

install_completions

echo "Installed $("$INSTALL_DIR/$BINARY" --version)"
