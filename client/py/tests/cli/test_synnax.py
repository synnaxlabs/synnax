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
import sys
from pathlib import Path

import pytest

from synnax.cli import parser


def run_sy(*args: str, home: Path) -> subprocess.CompletedProcess[str]:
    env = {
        **os.environ,
        "HOME": str(home),
        "USERPROFILE": str(home),
        "PYTHON_KEYRING_BACKEND": "keyring.backends.null.Keyring",
    }
    return subprocess.run(
        [sys.executable, "-c", "from synnax.cli.synnax import synnax; synnax()", *args],
        input="",
        capture_output=True,
        text=True,
        env=env,
        timeout=60,
    )


@pytest.mark.cli
class TestCommand:
    def test_requires_a_command(self) -> None:
        """Should exit with a usage error when no command is given."""
        with pytest.raises(SystemExit) as exc:
            parser().parse_args([])
        assert exc.value.code == 2

    def test_lists_login(self) -> None:
        """Should list login as a command."""
        assert "login" in parser().format_help()

    def test_login_fails_without_input(self, tmp_path: Path) -> None:
        """Should exit non-zero with one line and no traceback when input ends."""
        result = run_sy("login", home=tmp_path)
        assert result.returncode == 1
        assert "Traceback" not in result.stderr
        assert result.stderr.strip().splitlines()[-1] == "No input. Cancelled."
