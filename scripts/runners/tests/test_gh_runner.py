#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import threading
import time
from types import SimpleNamespace

import pytest
import synnax as sy

import conftest
import gh_runner
from conftest import core
from gh_runner import RETRY, SERVICE, TASK, Runner

CMD = "ubuntu_build_i_0f44_gh_cmd"
STATE = "ubuntu_build_i_0f44_gh_state"
KEY = 7
LISTENER = "Runner.Listener"
WORKER = "Runner.Worker"
START = ["systemctl", "start", SERVICE]
STOP = ["systemctl", "stop", SERVICE]


class Machine:
    """Stands in for the runner machine: lists its processes and records each service
    command. A command starts or stops the listener unless ignored is set."""

    def __init__(self) -> None:
        self.processes: set[str] = set()
        self.commands: list[list[str]] = []
        self.code = 0
        self.stderr = ""
        self.ignored = False
        self.error: Exception | None = None

    def process_iter(self, attrs: list[str]) -> list[SimpleNamespace]:
        return [SimpleNamespace(info={"name": name}) for name in self.processes]

    def run(self, cmd: list[str], **kwargs: object) -> SimpleNamespace:
        self.commands.append(cmd)
        if self.error is not None:
            raise self.error
        if not self.ignored:
            if any(part.lower().startswith("start") for part in cmd):
                self.processes.add(LISTENER)
            else:
                self.processes.discard(LISTENER)
        return SimpleNamespace(returncode=self.code, stdout="", stderr=self.stderr)


class Sink:
    def __init__(self) -> None:
        self.lines: list[str] = []

    def log(self, line: str) -> None:
        self.lines.append(line)


@pytest.fixture(autouse=True)
def machine(monkeypatch: pytest.MonkeyPatch) -> Machine:
    """Runs for every test, so none can reach the runner of the machine it runs on."""
    m = Machine()
    monkeypatch.setattr(gh_runner.psutil, "process_iter", m.process_iter)
    monkeypatch.setattr(gh_runner.subprocess, "run", m.run)
    monkeypatch.setattr(gh_runner, "sys", SimpleNamespace(platform="linux"))
    return m


@pytest.fixture
def sink() -> Sink:
    return Sink()


@pytest.fixture
def runner(sink: Sink) -> Runner:
    r = Runner()
    r.sink = sink
    return r


class TestApply:
    """Tests for starting and stopping the GitHub runner."""

    def test_should_start_the_runner(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        runner.apply(True, "command")
        assert machine.commands == [START]
        assert runner.state is True
        assert sink.lines == [
            "gh_cmd=True (command): starting runner",
            "runner started",
        ]

    def test_should_stop_the_runner(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.processes = {LISTENER}
        runner.apply(False, "command")
        assert machine.commands == [STOP]
        assert runner.state is False
        assert sink.lines[-1] == "runner stopped"

    def test_should_leave_a_runner_that_is_already_started(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.processes = {LISTENER}
        runner.apply(True, "stored")
        assert machine.commands == []
        assert runner.state is True
        assert sink.lines == ["gh_cmd=True (stored): runner already started"]

    def test_should_start_and_stop_the_task_on_windows(
        self, machine: Machine, runner: Runner, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(gh_runner, "sys", SimpleNamespace(platform="win32"))
        runner.apply(True, "command")
        runner.apply(False, "command")
        start, stop = machine.commands
        assert start[0] == stop[0] == "powershell"
        assert start[-1] == f"Start-ScheduledTask -TaskName '{TASK}'"
        assert stop[-1] == (
            f"Stop-ScheduledTask -TaskName '{TASK}' -ErrorAction SilentlyContinue; "
            "Stop-Process -Name 'Runner.Listener' -Force -ErrorAction SilentlyContinue"
        )
        assert runner.state is False

    def test_should_report_a_start_that_exits_with_an_error(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.code = 1
        machine.stderr = "Failed to start.\nUnit not found.\n"
        machine.ignored = True
        runner.apply(True, "command")
        assert runner.state is None
        assert sink.lines[-1] == "runner start failed rc=1: Unit not found."

    def test_should_report_a_stop_that_leaves_the_listener_up(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.processes = {LISTENER}
        machine.code = 1
        machine.stderr = "Access denied.\n"
        machine.ignored = True
        runner.apply(False, "command")
        assert runner.state is None
        assert sink.lines[-1] == "runner stop failed rc=1: Access denied."

    def test_should_accept_a_stop_with_an_error_code_when_the_listener_is_gone(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.processes = {LISTENER}
        machine.code = 1
        runner.apply(False, "command")
        assert runner.state is False
        assert sink.lines[-1] == "runner stopped"

    def test_should_report_a_command_that_cannot_run(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.error = FileNotFoundError("systemctl")
        runner.apply(True, "command")
        assert runner.state is None
        assert sink.lines[-1] == "runner start failed: FileNotFoundError: systemctl"

    def test_should_print_its_lines_without_a_sink(
        self, machine: Machine, capsys: pytest.CaptureFixture[str]
    ) -> None:
        machine.processes = {LISTENER}
        Runner().apply(True, "none yet")
        assert capsys.readouterr().out == (
            "gh_cmd=True (none yet): runner already started\n"
        )


class TestTick:
    """Tests for the work the runner control does on each sample."""

    def test_should_hold_a_stop_until_the_job_is_done(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        machine.processes = {LISTENER, WORKER}
        runner.apply(False, "command")
        runner.tick()
        assert machine.commands == []
        assert sink.lines == ["gh_cmd=False (command): job running, stopping after it"]
        machine.processes.discard(WORKER)
        runner.tick()
        assert machine.commands == [STOP]
        assert runner.state is False

    def test_should_start_a_runner_whose_listener_died(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        runner.apply(True, "command")
        machine.processes.clear()
        runner.attempted -= RETRY
        runner.tick()
        assert machine.commands == [START, START]
        assert "gh_cmd=True (not running): starting runner" in sink.lines

    def test_should_wait_before_it_starts_a_dead_listener_again(
        self, machine: Machine, runner: Runner
    ) -> None:
        runner.apply(True, "command")
        machine.processes.clear()
        runner.tick()
        assert machine.commands == [START]

    def test_should_leave_a_stopped_runner_stopped(
        self, machine: Machine, runner: Runner
    ) -> None:
        runner.apply(False, "command")
        runner.attempted -= RETRY
        runner.tick()
        assert machine.commands == []


class Streamer(conftest.Streamer):
    """Fails the next reads counted in errors, then streams the frames, then signals
    drained. The first streamer of a client with a hold waits in each read for it."""

    def __init__(self, client: "Core") -> None:
        super().__init__(client)
        self.core = client
        self.held = client.hold is not None and client.opened == 1

    def read(self, *, timeout: sy.TimeSpan) -> dict | None:
        if self.held and self.core.hold is not None:
            self.core.entered.set()
            self.core.hold.wait(5)
            time.sleep(0.001)
            return None
        if self.core.errors:
            self.core.errors -= 1
            raise ConnectionError("Core unreachable")
        if self.core.streamed:
            return self.core.streamed.pop(0)
        self.core.drained.set()
        time.sleep(0.001)
        return None

    def close(self) -> None:
        self.core.closed += 1


class Core(conftest.Client):
    """Adds the stored command states, found from the given lookback on, and the
    channel lookup."""

    def __init__(self) -> None:
        super().__init__()
        self.stored: list[bool] = []
        self.found_at = 1
        self.reads = 0
        self.errors = 0
        self.opened = 0
        self.closed = 0
        self.drained = threading.Event()
        self.hold: threading.Event | None = None
        self.entered = threading.Event()
        self.channels = SimpleNamespace(retrieve=lambda name: SimpleNamespace(key=KEY))

    def read(self, time_range: sy.TimeRange, name: str) -> list[bool]:
        self.reads += 1
        return self.stored if self.reads >= self.found_at else []

    def open_streamer(self, *args: object, **kwargs: object) -> Streamer:
        self.opened += 1
        return Streamer(self)


def listen(client: Core, runner: Runner, sink: Sink) -> None:
    """Listens until the client has no more frames to stream."""
    runner.listen(core(client), CMD, STATE, sink)
    assert client.drained.wait(5)
    runner.stop_listening()
    assert runner.thread is not None
    runner.thread.join(5)
    assert not runner.thread.is_alive()


class TestListen:
    """Tests for applying the stored state and the commands of the Core."""

    def test_should_apply_the_last_stored_state(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        client = Core()
        client.stored = [True, False]
        machine.processes = {LISTENER}
        listen(client, runner, sink)
        assert client.reads == 1
        assert machine.commands == [STOP]
        assert sink.lines[0] == "gh_cmd=False (stored): stopping runner"

    def test_should_look_further_back_for_a_stored_state(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        client = Core()
        client.stored = [False]
        client.found_at = 2
        listen(client, runner, sink)
        assert client.reads == 2
        assert sink.lines == ["gh_cmd=False (stored): runner already stopped"]

    def test_should_start_the_runner_without_a_stored_state(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        client = Core()
        listen(client, runner, sink)
        assert machine.commands == [START]
        assert sink.lines[0] == "gh_cmd=True (none yet): starting runner"

    @pytest.mark.parametrize("label", [CMD, KEY])
    def test_should_apply_a_command_under_the_name_or_the_key(
        self, machine: Machine, runner: Runner, sink: Sink, label: str | int
    ) -> None:
        client = Core()
        client.stored = [True]
        client.streamed = [{label: [True, False]}]
        machine.processes = {LISTENER}
        listen(client, runner, sink)
        assert machine.commands == [STOP]
        assert "gh_cmd=False (command): stopping runner" in sink.lines
        assert client.closed == 1

    def test_should_skip_a_frame_without_a_command(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        client = Core()
        client.stored = [True]
        client.streamed = [{"other": [False]}, {CMD: []}]
        machine.processes = {LISTENER}
        listen(client, runner, sink)
        assert machine.commands == []

    def test_should_open_a_new_streamer_after_a_failure(
        self,
        machine: Machine,
        runner: Runner,
        sink: Sink,
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        monkeypatch.setattr(gh_runner, "LISTEN_RETRY", 0)
        client = Core()
        client.stored = [True]
        client.streamed = [{CMD: [False]}]
        client.errors = 1
        machine.processes = {LISTENER}
        listen(client, runner, sink)
        assert client.opened == 2
        assert "commands failed: ConnectionError: Core unreachable" in sink.lines
        assert machine.commands == [STOP]

    def test_should_end_the_listener_of_the_session_before(
        self, machine: Machine, runner: Runner, sink: Sink
    ) -> None:
        client = Core()
        client.stored = [True]
        client.hold = threading.Event()
        machine.processes = {LISTENER}
        runner.listen(core(client), CMD, STATE, sink)
        old = runner.thread
        assert old is not None
        assert client.entered.wait(5)
        runner.stop_listening()
        runner.listen(core(client), CMD, STATE, sink)
        client.hold.set()
        old.join(5)
        assert not old.is_alive()
        listen_end = runner.thread
        runner.stop_listening()
        assert listen_end is not None
        listen_end.join(5)
        assert not listen_end.is_alive()
