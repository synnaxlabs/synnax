#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
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
        integrations: list[str] | None = None,
    ) -> None:
        """:param stop_timeout: Seconds stop waits for the Driver before it kills it.
        :param integrations: The integrations the Driver runs, such as the ones that
            ship dark. None runs its defaults except NI.
        """
        self._stop_timeout = stop_timeout
        self._state = Path(tempfile.mkdtemp(prefix="synnax-driver-"))
        secrets = self._state / "secrets"
        secrets.mkdir()
        config = self._state / "config.json"
        config.write_text(json.dumps({"secrets_dir": str(secrets)}))
        log_path.parent.mkdir(parents=True, exist_ok=True)
        self._log = open(log_path, "wb")
        args = [str(binary), "start", "--standalone", "--disable-sig-stop"]
        args += ["--host", connection.server_address]
        args += ["--port", str(connection.port)]
        args += ["--username", connection.username]
        args += ["--password", connection.password]
        args += [f"--secure={str(connection.secure).lower()}"]
        args += ["--state-file", str(self._state / "state.json")]
        args += ["--config", str(config)]
        if integrations is None:
            args += ["--disable-integrations", "ni"]
        else:
            args += ["--enable-integrations", ",".join(integrations)]
        # Unbuffered, so a write to a Driver that exited fails at once.
        self._popen = subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=self._log,
            stderr=subprocess.STDOUT,
            bufsize=0,
        )

    def api_key(self) -> str | None:
        """Returns the API key that the Driver logs in with as its rack, or None until
        the Driver enrolls with the Core."""
        try:
            state = json.loads((self._state / "state.json").read_text())
        # The Driver writes the file in place, so a read can see it half written.
        except (FileNotFoundError, json.JSONDecodeError):
            return None
        params = state.get("conn_params")
        if params is None:
            return None
        return json.loads(params).get("api_key") or None

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
