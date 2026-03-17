#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import subprocess


def get_synnax_version() -> str:
    """Get the current Synnax version from the VERSION file or git tags."""
    possible_paths = [
        "../core/pkg/version/VERSION",
        "core/pkg/version/VERSION",
        "../../../core/pkg/version/VERSION",
    ]

    for version_file in possible_paths:
        if os.path.exists(version_file):
            try:
                with open(version_file, "r") as f:
                    version = f.read().strip()
                    if version:
                        return version
            except (FileNotFoundError, PermissionError):
                continue

    result = subprocess.run(
        ["git", "describe", "--tags", "--abbrev=0"],
        capture_output=True,
        text=True,
        timeout=5,
    )
    if result.returncode == 0:
        version = result.stdout.strip()
        if version and version.startswith("v"):
            version = version[1:]
        if version:
            return version

    raise RuntimeError(
        "Unable to determine Synnax version from VERSION file or git tags"
    )
