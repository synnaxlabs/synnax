#!/bin/bash

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Installs the clang-format that CI formats with onto the uv tool PATH. Keep the version
# equal to the clang-tidy that MODULE.bazel pins.

set -euo pipefail

VERSION=23.1.3

uv tool install --force "clang-format==$VERSION"
