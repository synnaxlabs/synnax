#!/usr/bin/env zsh
# This is a script to kill any dangling tauri dev processes on macos. Hopefully tauri
# fixes these issues in teh future.

set -o localoptions -o localtraps
trap 'ps aux | grep "tauri dev" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9 && ps aux | grep "Synnax" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9' INT
sleep 100000
echo "returned with: $?"
