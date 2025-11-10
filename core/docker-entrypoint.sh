#!/bin/sh
set -e

# Parse flags to find data, certs, and log directories
DATA_DIR=""
CERTS_DIR=""
LOG_FILE_PATH=""

i=1
while [ $i -le $# ]; do
    eval arg=\${$i}
    case "$arg" in
        --data=*)
            DATA_DIR="${arg#*=}"
            ;;
        --data | -d)
            i=$((i + 1))
            eval DATA_DIR=\${$i}
            ;;
        --certs-dir=*)
            CERTS_DIR="${arg#*=}"
            ;;
        --certs-dir)
            i=$((i + 1))
            eval CERTS_DIR=\${$i}
            ;;
        --log-file-path=*)
            LOG_FILE_PATH="${arg#*=}"
            ;;
        --log-file-path)
            i=$((i + 1))
            eval LOG_FILE_PATH=\${$i}
            ;;
    esac
    i=$((i + 1))
done

# Use defaults if not specified
DATA_DIR="${DATA_DIR:-synnax-data}"
CERTS_DIR="${CERTS_DIR:-/usr/local/synnax/certs}"
LOG_FILE_PATH="${LOG_FILE_PATH:-./synnax-logs/synnax.log}"

# Extract log directory from log file path
LOG_DIR=$(dirname "$LOG_FILE_PATH")

# Ensure directories exist and fix ownership
mkdir -p "$DATA_DIR" "$CERTS_DIR" "$LOG_DIR" /home/nonroot/.cache /var/lib/synnax-driver 2> /dev/null || true
chown -R nonroot:nonroot "$DATA_DIR" "$CERTS_DIR" "$LOG_DIR" /home/nonroot/.cache /var/lib/synnax-driver 2> /dev/null || true

# Drop to nonroot user and execute command
exec gosu nonroot:nonroot "$@"
