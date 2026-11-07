#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The Driver built for NI Linux RT, run on the cRIO-9041 of the HITL station."""

import pathlib
import subprocess
import tempfile

import synnax as sy
from framework import resources
from framework.hardware_case import HardwareCase

HOST = "admin@192.168.50.30"
"""The cRIO on the instrument subnet. On NI Linux RT, admin is root."""
KEY = pathlib.Path("C:/hitl/keys/crio")
"""The key of the cRIO, which only the runner's user can read."""
KNOWN_HOSTS = pathlib.Path("C:/hitl/keys/known_hosts")
DRIVER = pathlib.Path.home() / "synnax-binaries" / "synnax-driver-nilinuxrt"
"""The Driver that the workflow downloads from the build."""
RACK_NAME = "NI-cRIO-9041-021061D4"
"""The rack the Driver enrolls, named after the cRIO's hostname."""

_DIR = "/home/admin/hitl"
_SSH_TIMEOUT = 60
_STATUS_TIMEOUT = 60 * sy.TimeSpan.SECOND
_SSH_OPTIONS = [
    "-i",
    str(KEY),
    "-o",
    "IdentitiesOnly=yes",
    "-o",
    "BatchMode=yes",
    "-o",
    f"UserKnownHostsFile={KNOWN_HOSTS}",
    "-o",
    "StrictHostKeyChecking=yes",
    "-o",
    "ConnectTimeout=10",
]


class CRIOCase(HardwareCase):
    """Runs the Driver on the cRIO for the length of a test.

    Setup sets the cRIO's clock from this machine and starts the Driver with a new state
    file, so the Driver enrolls a new rack on the Core under test. The cRIO dials the
    address the conductor connects to, so that address must reach the Core from the
    instrument subnet. Teardown stops the Driver, copies its log into the results, and
    deletes the rack.
    """

    rack: sy.Rack

    @classmethod
    def resources(cls) -> frozenset[str]:
        return super().resources() | {resources.CRIO}

    def setup(self) -> None:
        # The Driver stamps samples with the cRIO's clock, and nothing else syncs it. A
        # run that the conductor abandoned can leave its Driver running.
        self.ssh(
            f"date -u -s @{sy.TimeStamp.now() / 1e9:.3f} && "
            f"(pkill -9 -x synnax-driver; rm -rf {_DIR}) && "
            f"mkdir -p {_DIR}"
        )
        self._scp([str(DRIVER), f"{HOST}:{_DIR}/synnax-driver"])
        self.start_driver()
        self.rack = self.wait_for_rack("success")
        self.track_test_racks([self.rack])
        # Later setups in the chain, such as ArcCase's, deploy to the running Driver.
        super().setup()

    def start_driver(self) -> None:
        """Starts the Driver in the background with the state from the last start."""
        conn = self.synnax_connection
        # Only the Driver runs in the background, so the session ends and $! is its PID.
        self.ssh(
            f"cd {_DIR} && chmod +x synnax-driver && {{ nohup ./synnax-driver start "
            f"--standalone --host {conn.server_address} --port {conn.port} "
            f"--username {conn.username} --password {conn.password} "
            f"--state-file {_DIR}/state.json --disable-stdin-stop "
            f">> driver.log 2>&1 < /dev/null & echo $! > driver.pid; }}"
        )

    def stop_driver(self) -> None:
        """Stops the Driver and waits for it to exit."""
        self.ssh(
            f"pid=$(cat {_DIR}/driver.pid) && kill $pid; "
            "for i in $(seq 50); do kill -0 $pid 2>/dev/null || exit 0; sleep 0.1; "
            "done; kill -9 $pid"
        )

    def wait_for_rack(self, variant: str) -> sy.Rack:
        """Returns the cRIO's rack once its status has the variant. Success means the
        Driver is running, and a warning means the Core stopped hearing from it.

        :raises TimeoutError: If no rack reaches the variant within 60 seconds.
        """
        timer = sy.Timer()
        while True:
            for rack in self.client.racks.retrieve(
                names=[RACK_NAME], include_status=True
            ):
                if rack.status is not None and rack.status.variant == variant:
                    return rack
            if timer.elapsed() > _STATUS_TIMEOUT:
                raise TimeoutError(f"Rack {RACK_NAME} did not reach {variant}")
            sy.sleep(0.5)

    def ssh(self, command: str) -> None:
        """Runs a shell command on the cRIO.

        :raises RuntimeError: If the command fails.
        """
        _run(["ssh", *_SSH_OPTIONS, HOST, command])

    def teardown(self) -> None:
        try:
            with self._try_to("stop the cRIO Driver"):
                self.stop_driver()
            with self._try_to("copy the cRIO Driver log"):
                log = self.results_dir / "crio-driver.log"
                self._scp([f"{HOST}:{_DIR}/driver.log", str(log)])
            # Setup can fail before the Driver enrolls its rack.
            rack = getattr(self, "rack", None)
            if rack is not None:
                with self._try_to("track the cRIO devices"):
                    self.track_test_devices(self.client.devices.retrieve(rack=rack.key))
        finally:
            super().teardown()

    def _scp(self, paths: list[str]) -> None:
        _run(["scp", "-q", *_SSH_OPTIONS, *paths])


def _run(args: list[str]) -> None:
    # Windows OpenSSH does not exit while its output is a pipe, so it writes to a file.
    with tempfile.TemporaryFile("w+") as output:
        result = subprocess.run(
            args,
            stdin=subprocess.DEVNULL,
            stdout=output,
            stderr=subprocess.STDOUT,
            timeout=_SSH_TIMEOUT,
        )
        if result.returncode != 0:
            output.seek(0)
            raise RuntimeError(
                f"{args[0]} exited with code {result.returncode}: {output.read()}"
            )
