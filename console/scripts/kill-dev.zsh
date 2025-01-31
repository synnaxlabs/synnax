#!/usr/bin/env zsh
#
# Copyright 2024 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.
#

# This is a script to kill any dangling tauri dev processes on macOS. Hopefully tauri
# fixes these issues in the future.

set -o localoptions -o localtraps
trap 'ps aux | grep "tauri.js dev" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9 && ps aux | grep "Synnax" | grep -v grep | awk '"'"'{print $2}'"'"' | xargs kill -9' INT
sleep 100000
echo "returned with: $?"
