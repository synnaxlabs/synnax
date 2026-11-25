#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import importlib.metadata
import subprocess
import sys

import synnax as sy

# Runs the client as a frozen app does, where no dist-info for synnax exists.
_WITHOUT_METADATA = """
import importlib.metadata
import sys

_version = importlib.metadata.version


def _without_synnax(name):
    if name == "synnax":
        raise importlib.metadata.PackageNotFoundError(name)
    return _version(name)


importlib.metadata.version = _without_synnax

import synnax as sy

host, port, username, password = sys.argv[1:]
client = sy.Synnax(
    host=host, port=int(port), username=username, password=password, secure=False
)
client.close()
print(sy.__version__)
"""


class TestVersion:
    """The version of the client."""

    def test_matches_package_metadata(self) -> None:
        """Should equal the version in the package metadata."""
        assert sy.__version__ == importlib.metadata.version("synnax")

    def test_connect_without_package_metadata(
        self, login_info: tuple[str, int, str, str]
    ) -> None:
        """Should import the client and connect when the package metadata is missing,
        as in a PyInstaller app."""
        host, port, username, password = login_info
        res = subprocess.run(
            [
                sys.executable,
                "-c",
                _WITHOUT_METADATA,
                host,
                str(port),
                username,
                password,
            ],
            capture_output=True,
            text=True,
            timeout=60,
            check=False,
        )
        assert res.returncode == 0, res.stderr
        assert res.stdout.strip() == sy.__version__
