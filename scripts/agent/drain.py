#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Drains a seat whose login reached its usage limit. Each runner service stops as
soon as it runs no job, so it takes no new work, and a job that runs finishes. The
instance powers off once every runner has stopped.

Usage: python3 drain.py, as root, outside the runner services (systemd-run).
"""

import subprocess
import time
from collections.abc import Callable
from pathlib import Path
from typing import Protocol

WORKER = "Runner.Worker"
INTERVAL = 1.0


class Services(Protocol):
    """The runner services drain() stops and the power off after them."""

    def units(self) -> set[str]: ...

    def busy(self) -> set[str]: ...

    def stop(self, unit: str) -> None: ...

    def poweroff(self) -> None: ...


class System:
    """The runner services and processes of this instance, through systemd and
    procfs."""

    def __init__(self, *, proc: Path = Path("/proc")) -> None:
        self.proc = proc

    def units(self) -> set[str]:
        """The runner services that run now."""
        cmd = ["systemctl", "list-units", "actions.runner.*", "--plain"]
        cmd += ["--no-legend", "--state=active"]
        out = subprocess.run(cmd, check=True, text=True, capture_output=True).stdout
        return {line.split()[0] for line in out.splitlines() if line.strip()}

    def busy(self) -> set[str]:
        """The services that run a job, found from the cgroup of each job worker."""
        units: set[str] = set()
        for pid in self.proc.iterdir():
            if not pid.name.isdecimal():
                continue
            try:
                if (pid / "comm").read_text().strip() != WORKER:
                    continue
                cgroup = (pid / "cgroup").read_text()
            except (FileNotFoundError, ProcessLookupError):
                # The process exited.
                continue
            units.update(line.rsplit("/", 1)[-1] for line in cgroup.splitlines())
        return units

    def stop(self, unit: str) -> None:
        subprocess.run(["systemctl", "stop", unit], check=True)

    def poweroff(self) -> None:
        subprocess.run(["systemctl", "poweroff"], check=True)


def drain(system: Services, *, sleep: Callable[[float], None] = time.sleep) -> None:
    """Stops each idle runner service until none runs, then powers off."""
    while units := system.units():
        # A stop can take seconds, so each runner is checked right before its stop.
        for unit in sorted(units):
            if unit not in system.busy():
                system.stop(unit)
        sleep(INTERVAL)
    system.poweroff()


if __name__ == "__main__":
    drain(System())
