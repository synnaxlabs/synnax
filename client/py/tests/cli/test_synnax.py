#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import signal
import socket
import subprocess
import sys
from pathlib import Path

import pytest

from synnax.cli import parser

_RUN_SY = "from synnax.cli.synnax import synnax; synnax()"


def sy_env(home: Path) -> dict[str, str]:
    return {
        **os.environ,
        "HOME": str(home),
        "USERPROFILE": str(home),
        "PYTHON_KEYRING_BACKEND": "keyring.backends.null.Keyring",
    }


def run_sy(*args: str, home: Path, input: str = "") -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        [sys.executable, "-c", _RUN_SY, *args],
        input=input,
        capture_output=True,
        text=True,
        env=sy_env(home),
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

    def test_login_fails_when_core_is_unreachable(self, tmp_path: Path) -> None:
        """Should exit with 1 and one line when no Core listens at the address."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = s.getsockname()[1]
        result = run_sy(
            "login", home=tmp_path, input=f"127.0.0.1\n{port}\nsynnax\nsecret\nn\n"
        )
        assert result.returncode == 1
        assert "Traceback" not in result.stderr
        assert result.stderr.strip().splitlines()[-1] == (
            f"Cannot reach Synnax server at 127.0.0.1:{port}"
        )

    @pytest.mark.skipif(sys.platform == "win32", reason="SIGINT needs a POSIX signal")
    def test_login_exits_130_on_interrupt(self, tmp_path: Path) -> None:
        """Should exit with 130 and no traceback when the user presses Ctrl+C."""
        with subprocess.Popen(
            [sys.executable, "-c", _RUN_SY, "login"],
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            env=sy_env(tmp_path),
        ) as proc:
            assert proc.stdout is not None
            assert proc.stderr is not None
            # The first prompt line shows that the command waits for input.
            proc.stdout.readline()
            # stdin stays open until the exit, or the command can read end of input
            # before it handles the signal. A SIGINT that lands before input() starts
            # its read only sets a flag, and the read then blocks, so it is sent again
            # until the command exits, as a user presses Ctrl+C again. The long gap
            # keeps a second SIGINT away from a slow exit.
            for _ in range(12):
                proc.send_signal(signal.SIGINT)
                try:
                    proc.wait(timeout=5)
                    break
                except subprocess.TimeoutExpired:
                    continue
            else:
                proc.kill()
                proc.wait()
            stderr = proc.stderr.read()
        assert proc.returncode == 130
        assert "Traceback" not in stderr
