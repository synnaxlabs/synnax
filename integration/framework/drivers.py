#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import shutil
import subprocess
import tempfile
from pathlib import Path

from framework.models import SynnaxConnection

_STOP_TIMEOUT = 15.0


class Process:
    """A standalone Driver that runs apart from the Core it connects to, so it
    outlives a restart of that Core."""

    def __init__(
        self,
        *,
        binary: Path,
        connection: SynnaxConnection,
        log_path: Path,
        stop_timeout: float = _STOP_TIMEOUT,
    ) -> None:
        """:param stop_timeout: Seconds stop waits for the Driver before it kills it."""
        self._stop_timeout = stop_timeout
        self._state = Path(tempfile.mkdtemp(prefix="synnax-driver-"))
        log_path.parent.mkdir(parents=True, exist_ok=True)
        self._log = open(log_path, "wb")
        args = [str(binary), "start", "--standalone", "--disable-sig-stop"]
        args += ["--host", connection.server_address]
        args += ["--port", str(connection.port)]
        args += ["--username", connection.username]
        args += ["--password", connection.password]
        args += [f"--secure={str(connection.secure).lower()}"]
        args += ["--state-file", str(self._state / "state.json")]
        args += ["--disable-integrations", "ni"]
        # Unbuffered, so a write to a Driver that exited fails at once.
        self._popen = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=self._log,
            stderr=subprocess.STDOUT,
            bufsize=0,
        )

    def stop(self) -> None:
        """Asks the Driver to stop its tasks and exit, then kills a Driver that does
        not exit in time."""
        stdin = self._popen.stdin
        assert stdin is not None
        try:
            stdin.write(b"STOP\n")
        except OSError:
            # The Driver already exited. Windows raises EINVAL, not a broken pipe.
            pass
        stdin.close()
        try:
            self._popen.wait(timeout=self._stop_timeout)
        except subprocess.TimeoutExpired:
            self._popen.kill()
            self._popen.wait()
        self._log.close()
        shutil.rmtree(self._state, ignore_errors=True)
