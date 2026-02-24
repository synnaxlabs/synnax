#!/bin/bash
# Copyright 2025 Synnax Labs, Inc.
#
# Oracle Installation Script
# Builds oracle, installs the Cursor/VSCode LSP extension, and adds oracle to PATH.

set -e

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Configuration
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ORACLE_ROOT="$SCRIPT_DIR"
EXTENSION_DIR="$ORACLE_ROOT/lsp/extensions/vscode"
INSTALL_DIR="$HOME/.local/bin"
LOCK_FILE="/tmp/oracle-install.lock"
START_TIME=$(date +%s)

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Colors & Styling
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

NC='\033[0m'
BOLD='\033[1m'
DIM='\033[2m'

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
WHITE='\033[0;37m'

BOLD_GREEN='\033[1;32m'
BOLD_CYAN='\033[1;36m'
BOLD_WHITE='\033[1;37m'
BOLD_YELLOW='\033[1;33m'

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Helper Functions
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

header() {
    printf "\n"
    printf "  ${DIM}╭────────────────────────────────────────────────────────────╮${NC}\n"
    printf "  ${DIM}│${NC}                                                            ${DIM}│${NC}\n"
    printf "  ${DIM}│${NC}   ${BOLD_WHITE}◆  Oracle Installer${NC}                                      ${DIM}│${NC}\n"
    printf "  ${DIM}│${NC}                                                            ${DIM}│${NC}\n"
    printf "  ${DIM}╰────────────────────────────────────────────────────────────╯${NC}\n"
    printf "\n"
}

section() {
    printf "\n  ${BOLD}$1${NC}\n"
    printf "  ${DIM}────────────────────────────────────────────────────────────${NC}\n"
}

step() {
    printf "\n  ${BOLD_CYAN}[$1/$2]${NC} ${BOLD}$3${NC}\n"
}

ok() {
    printf "       ${GREEN}✓${NC}  $1\n"
}

info() {
    printf "       ${BLUE}›${NC}  ${DIM}$1${NC}\n"
}

warn() {
    printf "       ${YELLOW}!${NC}  $1\n"
}

fail() {
    printf "       ${RED}✗${NC}  $1\n"
}

spinner() {
    local pid=$1
    local msg=$2
    local frames='⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏'
    local i=0

    while kill -0 "$pid" 2> /dev/null; do
        printf "\r       ${CYAN}${frames:i++%${#frames}:1}${NC}  ${DIM}$msg${NC}"
        sleep 0.08
    done
    printf "\r\033[K"
}

run() {
    local msg=$1
    shift
    local logfile
    logfile=$(mktemp /tmp/oracle-install-XXXXXX.log)
    "$@" > "$logfile" 2>&1 &
    local pid=$!
    spinner $pid "$msg"
    local exit_code=0
    wait $pid || exit_code=$?
    if [ $exit_code -ne 0 ]; then
        fail "Command failed (exit $exit_code): $*"
        printf "\n"
        info "Output:"
        while IFS= read -r line; do
            printf "       ${DIM}  %s${NC}\n" "$line"
        done < "$logfile"
        printf "\n"
        rm -f "$logfile"
        exit $exit_code
    fi
    rm -f "$logfile"
}

elapsed() {
    local now=$(date +%s)
    local diff=$((now - START_TIME))
    if [ $diff -lt 60 ]; then
        echo "${diff}s"
    else
        echo "$((diff / 60))m $((diff % 60))s"
    fi
}

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Lock Management
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

acquire_lock() {
    if [ -f "$LOCK_FILE" ]; then
        local pid=$(cat "$LOCK_FILE" 2> /dev/null)
        if [ -n "$pid" ] && kill -0 "$pid" 2> /dev/null; then
            fail "Another installation is running (PID: $pid)"
            exit 1
        fi
        rm -f "$LOCK_FILE"
    fi
    echo $$ > "$LOCK_FILE"
}

release_lock() {
    rm -f "$LOCK_FILE"
}

trap release_lock EXIT

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Help
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

show_help() {
    printf "\n"
    printf "  ${BOLD}Oracle Installer${NC}\n"
    printf "  ${DIM}Build and install the Oracle CLI and editor extension${NC}\n"
    printf "\n"
    printf "  ${BOLD}Usage${NC}\n"
    printf "      ./install.sh ${DIM}[options]${NC}\n"
    printf "\n"
    printf "  ${BOLD}Options${NC}\n"
    printf "      ${BOLD}--cli-only${NC}            Build CLI only, skip extension\n"
    printf "      ${BOLD}--extension-only${NC}      Install extension only, skip CLI\n"
    printf "      ${BOLD}--no-path${NC}             Don't modify shell PATH\n"
    printf "      ${BOLD}--install-dir${NC} ${DIM}<dir>${NC}   Install location ${DIM}(default: ~/.local/bin)${NC}\n"
    printf "      ${BOLD}-h, --help${NC}            Show this help\n"
    printf "\n"
    printf "  ${BOLD}Examples${NC}\n"
    printf "      ${DIM}# Full installation${NC}\n"
    printf "      ./install.sh\n"
    printf "\n"
    printf "      ${DIM}# CLI only, custom location${NC}\n"
    printf "      ./install.sh --cli-only --install-dir /usr/local/bin\n"
    printf "\n"
    exit 0
}

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Parse Arguments
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

INSTALL_CLI=true
INSTALL_EXTENSION=true
ADD_TO_PATH=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --cli-only)
            INSTALL_EXTENSION=false
            shift
            ;;
        --extension-only)
            INSTALL_CLI=false
            ADD_TO_PATH=false
            shift
            ;;
        --no-path)
            ADD_TO_PATH=false
            shift
            ;;
        --install-dir)
            INSTALL_DIR="$2"
            shift 2
            ;;
        -h | --help)
            show_help
            ;;
        *)
            fail "Unknown option: $1"
            info "Run ./install.sh --help for usage"
            exit 1
            ;;
    esac
done

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Calculate Steps
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

TOTAL=0
$INSTALL_CLI && ((TOTAL += 1))
$ADD_TO_PATH && ((TOTAL += 1))
$INSTALL_EXTENSION && ((TOTAL += 4))
STEP=0

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Main
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

header
acquire_lock

# Prerequisites
section "Prerequisites"

if ! command -v go &> /dev/null; then
    fail "Go not found"
    info "Install from https://go.dev/dl"
    exit 1
fi
ok "Go $(go version | awk '{print $3}' | sed 's/go//')"

if $INSTALL_EXTENSION; then
    if ! command -v npm &> /dev/null; then
        fail "npm not found"
        info "Install from https://nodejs.org"
        exit 1
    fi
    ok "npm $(npm --version)"
fi

# Build CLI
if $INSTALL_CLI; then
    ((STEP += 1))
    step $STEP $TOTAL "Build CLI"

    cd "$ORACLE_ROOT"
    mkdir -p "$INSTALL_DIR"

    BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
    run "Compiling..." go build \
        -ldflags "-X 'github.com/synnaxlabs/oracle/cmd.BuildTime=$BUILD_TIME'" \
        -o "$INSTALL_DIR/oracle" .

    chmod +x "$INSTALL_DIR/oracle"
    ok "Built → ${DIM}$INSTALL_DIR/oracle${NC}"

    if $INSTALL_EXTENSION; then
        mkdir -p "$EXTENSION_DIR/bin"
        cp "$INSTALL_DIR/oracle" "$EXTENSION_DIR/bin/oracle"
        ok "Copied → ${DIM}extension/bin/oracle${NC}"
    fi
fi

# Configure PATH
if $ADD_TO_PATH; then
    ((STEP += 1))
    step $STEP $TOTAL "Configure PATH"

    SHELL_CONFIG=""
    [[ "$SHELL" == *"zsh"* ]] && SHELL_CONFIG="$HOME/.zshrc"
    [[ "$SHELL" == *"bash"* ]] && SHELL_CONFIG="${HOME}/.bash_profile"
    [[ -z "$SHELL_CONFIG" && -f "$HOME/.bashrc" ]] && SHELL_CONFIG="$HOME/.bashrc"

    PATH_LINE="export PATH=\"$INSTALL_DIR:\$PATH\""

    if [ -n "$SHELL_CONFIG" ]; then
        if grep -q "$INSTALL_DIR" "$SHELL_CONFIG" 2> /dev/null; then
            ok "Already in PATH"
            info "$SHELL_CONFIG"
        else
            printf "\n# Oracle CLI\n%s\n" "$PATH_LINE" >> "$SHELL_CONFIG"
            ok "Added to $SHELL_CONFIG"
        fi
    else
        warn "Shell config not found"
        info "Add manually: $PATH_LINE"
    fi

    export PATH="$INSTALL_DIR:$PATH"
fi

# Extension
if $INSTALL_EXTENSION; then
    cd "$EXTENSION_DIR"

    ((STEP += 1))
    step $STEP $TOTAL "Install Dependencies"
    run "npm install..." npm install --silent
    ok "Dependencies ready"

    ((STEP += 1))
    step $STEP $TOTAL "Compile Extension"
    run "Compiling TypeScript..." npm run compile --silent
    ok "Compiled"

    ((STEP += 1))
    step $STEP $TOTAL "Package Extension"
    vsix_log=$(mktemp /tmp/oracle-install-XXXXXX.log)
    if ! yes 2> /dev/null | npx @vscode/vsce package \
        --allow-missing-repository -o oracle-language.vsix > "$vsix_log" 2>&1; then
        fail "Failed to create package"
        printf "\n"
        info "Output:"
        while IFS= read -r line; do
            printf "       ${DIM}  %s${NC}\n" "$line"
        done < "$vsix_log"
        printf "\n"
        rm -f "$vsix_log"
        exit 1
    fi
    rm -f "$vsix_log"

    if [ -f "oracle-language.vsix" ]; then
        SIZE=$(du -h oracle-language.vsix | cut -f1 | tr -d ' ')
        ok "Created → ${DIM}oracle-language.vsix ($SIZE)${NC}"
    else
        fail "Failed to create package (no output file)"
        exit 1
    fi

    ((STEP += 1))
    step $STEP $TOTAL "Install to Editors"

    EDITORS=()

    # Cursor
    CURSOR=""
    command -v cursor &> /dev/null && CURSOR="cursor"
    [ -z "$CURSOR" ] && [ -f "/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ] \
        && CURSOR="/Applications/Cursor.app/Contents/Resources/app/bin/cursor"
    [ -z "$CURSOR" ] && [ -f "$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor" ] \
        && CURSOR="$HOME/Applications/Cursor.app/Contents/Resources/app/bin/cursor"

    if [ -n "$CURSOR" ]; then
        "$CURSOR" --install-extension "$EXTENSION_DIR/oracle-language.vsix" &> /dev/null \
            && EDITORS+=("Cursor")
    fi

    # VS Code
    VSCODE=""
    command -v code &> /dev/null && VSCODE="code"
    [ -z "$VSCODE" ] && [ -f "/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code" ] \
        && VSCODE="/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"

    if [ -n "$VSCODE" ]; then
        "$VSCODE" --install-extension "$EXTENSION_DIR/oracle-language.vsix" &> /dev/null \
            && EDITORS+=("VS Code")
    fi

    if [ ${#EDITORS[@]} -gt 0 ]; then
        for e in "${EDITORS[@]}"; do
            ok "Installed → ${DIM}$e${NC}"
        done
    else
        warn "No editors found"
        info "Install manually: Extensions → Install from VSIX"
    fi
fi

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# Summary
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

printf "\n"
printf "  ${DIM}────────────────────────────────────────────────────────────${NC}\n"
printf "  ${GREEN}✓${NC}  ${BOLD}Done${NC} ${DIM}in $(elapsed)${NC}\n"
printf "\n"

if $INSTALL_CLI; then
    printf "  ${DIM}CLI${NC}        $INSTALL_DIR/oracle\n"
fi
if $INSTALL_EXTENSION && [ -f "$EXTENSION_DIR/oracle-language.vsix" ]; then
    printf "  ${DIM}Extension${NC}  oracle-language.vsix\n"
fi

printf "\n"
printf "  ${DIM}Get started:${NC}\n"
printf "  ${CYAN}\$${NC} oracle --help\n"
printf "  ${CYAN}\$${NC} oracle sync\n"

if $ADD_TO_PATH && [ -n "$SHELL_CONFIG" ]; then
    printf "\n"
    printf "  ${DIM}Reload shell:${NC}\n"
    printf "  ${CYAN}\$${NC} source $SHELL_CONFIG\n"
fi

if $INSTALL_EXTENSION; then
    printf "\n"
    printf "  ${DIM}Restart your editor to activate the extension${NC}\n"
fi

printf "\n"
