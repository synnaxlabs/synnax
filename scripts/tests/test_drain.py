#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from typing import cast

from agent.drain import System, drain

UNIT = "actions.runner.synnaxlabs.agent-claude-1-{}.service"


def process(proc: Path, pid: int, *, comm: str, unit: str) -> None:
    d = proc / str(pid)
    d.mkdir(parents=True)
    (d / "comm").write_text(f"{comm}\n")
    (d / "cgroup").write_text(f"0::/system.slice/{unit}\n")


class Seat:
    """Stands in for System: each runner runs a job for a number of polls."""

    def __init__(self, jobs: dict[str, int]) -> None:
        self.jobs = jobs
        self.running = set(jobs)
        self.stopped: list[str] = []
        self.off = False

    def units(self) -> set[str]:
        return set(self.running)

    def busy(self) -> set[str]:
        return {u for u in self.running if self.jobs[u] > 0}

    def stop(self, unit: str) -> None:
        assert unit not in self.busy(), f"{unit} stopped during a job"
        self.running.remove(unit)
        self.stopped.append(unit)

    def poweroff(self) -> None:
        self.off = True

    def tick(self, _: float) -> None:
        self.jobs = {u: n - 1 for u, n in self.jobs.items()}


class TestSystem:
    """Tests for finding the runner services that run a job."""

    def test_should_find_the_services_of_job_workers(self, tmp_path: Path) -> None:
        process(tmp_path, 10, comm="Runner.Worker", unit=UNIT.format(1))
        process(tmp_path, 11, comm="Runner.Listener", unit=UNIT.format(2))
        process(tmp_path, 12, comm="bash", unit=UNIT.format(3))
        (tmp_path / "meminfo").write_text("MemTotal: 1 kB\n")
        assert System(proc=tmp_path).busy() == {UNIT.format(1)}


class TestDrain:
    """Tests for stopping the runners of a limited seat."""

    def test_should_stop_idle_runners_at_once(self) -> None:
        seat = Seat({UNIT.format(1): 0, UNIT.format(2): 3})
        drain(cast(System, seat), sleep=seat.tick)
        assert seat.stopped == [UNIT.format(1), UNIT.format(2)]

    def test_should_power_off_once_every_job_finished(self) -> None:
        seat = Seat({UNIT.format(1): 2, UNIT.format(2): 5})

        def tick(seconds: float) -> None:
            assert not seat.off
            seat.tick(seconds)

        drain(cast(System, seat), sleep=tick)
        assert seat.off
        assert seat.stopped == [UNIT.format(1), UNIT.format(2)]
