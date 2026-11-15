#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse
import json
import sys
from collections.abc import Iterable
from pathlib import Path
from types import SimpleNamespace
from typing import cast

import numpy as np
import psutil
import pytest
import synnax as sy

import agent
import agent_log
import common
import conftest
from agent import (
    API_KEY_PARAMETER,
    REGISTER_RETRY,
    REOPEN_DELAY,
    RETRY,
    SESSION_DELAY,
    already_running,
    channel_name,
    close_quietly,
    connect,
    deprioritize,
    run,
    runner_name,
)
from agent_log import Log
from gh_runner import Runner

RUNNER = "ubuntu-build-bot-i-0f44"
NAMES = {
    "index": "ubuntu_build_i_0f44_sys_time",
    "cpu": "ubuntu_build_i_0f44_cpu",
    "mem": "ubuntu_build_i_0f44_mem",
    "disk": "ubuntu_build_i_0f44_disk_free",
    "disk_percent": "ubuntu_build_i_0f44_disk_percent",
    "log": "ubuntu_build_i_0f44_log",
    "cmd": "ubuntu_build_i_0f44_gh_cmd",
    "state": "ubuntu_build_i_0f44_gh_state",
}
USAGE = [NAMES[k] for k in ("index", "cpu", "mem", "disk", "disk_percent")]


def arguments(*, secure: bool = False, no_runner: bool = False) -> argparse.Namespace:
    return argparse.Namespace(
        host="core.example", port=9090, secure=secure, no_runner=no_runner
    )


class Writer(conftest.Writer):
    def write(self, frame: dict) -> None:
        core = cast(Core, self.client)
        if rejected := core.rejected & frame.keys():
            core.rejected -= rejected
            raise ConnectionError("Core unreachable")
        super().write(frame)


class Core(conftest.Client):
    """Adds the channel lookup and the close of a Synnax client. The first write to a
    channel in rejected fails."""

    def __init__(self, names: Iterable[str] = NAMES.values()) -> None:
        super().__init__()
        self.names = set(names)
        self.rejected: set[str] = set()
        self.writers = 0
        self.closed = False
        self.channels = SimpleNamespace(retrieve=self.retrieve)

    def retrieve(self, names: list[str]) -> list[SimpleNamespace]:
        return [SimpleNamespace(name=n) for n in names if n in self.names]

    def open_writer(
        self, start: sy.TimeStamp, *args: object, **kwargs: object
    ) -> Writer:
        self.writers += 1
        return Writer(client=self, start=start)

    def close(self) -> None:
        self.closed = True


class Cores:
    """Stands in for sy.Synnax: hands out the given clients in order, raises the given
    errors, and records the arguments of each call."""

    def __init__(self, *clients: Core | Exception) -> None:
        self.clients = list(clients)
        self.calls: list[dict] = []

    def __call__(self, **kwargs: object) -> Core:
        self.calls.append(kwargs)
        client = self.clients.pop(0)
        if isinstance(client, Exception):
            raise client
        return client


class Store:
    """Stands in for Parameter Store: serves the API key after the reads counted in
    errors fail."""

    def __init__(self) -> None:
        self.requests: list[tuple[str, bool]] = []
        self.errors = 0

    def get_parameter(self, *, Name: str, WithDecryption: bool = False) -> dict:
        self.requests.append((Name, WithDecryption))
        if self.errors:
            self.errors -= 1
            raise ConnectionError("Parameter Store unreachable")
        return {"Parameter": {"Value": "syk_test"}}


class Control:
    """Stands in for the runner control and records what the agent asks of it."""

    def __init__(self) -> None:
        self.state: bool | None = None
        self.applied: list[tuple[bool, str]] = []
        self.listened: list[tuple[str, str]] = []
        self.sinks: list[object] = []
        self.sink: object = None
        self.ticks = 0
        self.listening = False

    def apply(self, enabled: bool, why: str) -> None:
        self.applied.append((enabled, why))
        self.state = enabled

    def listen(self, client: object, name: str, state: str, sink: object) -> None:
        self.listened.append((name, state))
        self.sinks.append(sink)
        self.sink = sink
        self.listening = True

    def tick(self) -> None:
        self.ticks += 1

    def stop_listening(self) -> None:
        self.listening = False


def control(c: Control) -> Runner:
    return cast(Runner, c)


class Loop:
    """Stands in for sy.Loop: runs the given number of passes without a wait."""

    def __init__(self, passes: int) -> None:
        self.passes = passes

    def wait(self) -> bool:
        self.passes -= 1
        return self.passes >= 0


@pytest.fixture
def sleeps(monkeypatch: pytest.MonkeyPatch) -> list[sy.TimeSpan]:
    """The spans the agent sleeps for, which take no time."""
    spans: list[sy.TimeSpan] = []
    monkeypatch.setattr(sy, "sleep", spans.append)
    return spans


@pytest.fixture
def store() -> Store:
    return Store()


def cores(monkeypatch: pytest.MonkeyPatch, *clients: Core | Exception) -> Cores:
    c = Cores(*clients)
    monkeypatch.setattr(sy, "Synnax", c)
    return c


class TestChannelName:
    """Tests for naming the channels of a runner."""

    def test_should_shorten_the_runner_name(self) -> None:
        assert channel_name("cpu", "windows-build-bot-i-0f44") == "win_build_i_0f44_cpu"

    @pytest.mark.parametrize(
        "runner", [RUNNER, "windows-test-bot-i-0b12", "macos-build-bot", "tester 1"]
    )
    def test_should_match_the_names_the_controller_creates(self, runner: str) -> None:
        for source in ("sys_time", "cpu", "log", "gh_cmd"):
            assert channel_name(source, runner) == common.channel_name(
                source, runner=runner
            )


class Broken:
    def __init__(self) -> None:
        self.closes = 0

    def close(self) -> None:
        self.closes += 1
        raise ConnectionError("Core unreachable")


class TestCloseQuietly:
    """Tests for closing a resource during teardown."""

    def test_should_ignore_a_close_that_fails(self) -> None:
        resource = Broken()
        close_quietly(resource)
        close_quietly(None)
        assert resource.closes == 1


@pytest.fixture
def registration(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    """The .runner file of the machine, absent until a test writes it. The name cache
    sits beside it and the wait for the file is over at once."""
    file = tmp_path / ".runner"
    monkeypatch.setattr(agent, "RUNNER_FILES", (tmp_path / "none" / ".runner", file))
    monkeypatch.setattr(agent, "NAME_CACHE", tmp_path / "runner_name.txt")
    monkeypatch.setattr(agent, "REGISTER_WAIT", sy.TimeSpan(0))
    return file


def register(file: Path) -> None:
    file.write_text(json.dumps({"agentName": RUNNER}), encoding="utf-8-sig")


class TestRunnerName:
    """Tests for finding the name of the runner on this machine."""

    def test_should_read_the_name_from_the_runner_file(
        self, registration: Path
    ) -> None:
        register(registration)
        assert runner_name() == RUNNER
        assert agent.NAME_CACHE.read_text() == RUNNER

    def test_should_use_the_cached_name_when_the_file_is_gone(
        self, registration: Path
    ) -> None:
        agent.NAME_CACHE.write_text(f"{RUNNER}\n")
        assert runner_name() == RUNNER

    @pytest.mark.parametrize("content", ["{}", "not json"])
    def test_should_skip_a_file_without_a_name(
        self, registration: Path, content: str
    ) -> None:
        registration.write_text(content)
        assert runner_name() is None

    def test_should_return_the_name_when_the_cache_cannot_be_written(
        self,
        registration: Path,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        register(registration)
        monkeypatch.setattr(agent, "NAME_CACHE", registration.parent / "none" / "name")
        assert runner_name() == RUNNER
        assert capsys.readouterr().out.startswith("name cache failed: FileNotFound")

    def test_should_wait_for_a_runner_that_still_registers(
        self, registration: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        spans: list[sy.TimeSpan] = []

        def sleep(span: sy.TimeSpan) -> None:
            spans.append(span)
            register(registration)

        monkeypatch.setattr(agent, "REGISTER_WAIT", sy.TimeSpan.MINUTE)
        monkeypatch.setattr(sy, "sleep", sleep)
        assert runner_name() == RUNNER
        assert spans == [REGISTER_RETRY]


def process(pid: int, name: str | None, *cmdline: str) -> SimpleNamespace:
    return SimpleNamespace(info={"pid": pid, "name": name, "cmdline": list(cmdline)})


@pytest.fixture
def processes(monkeypatch: pytest.MonkeyPatch) -> list[SimpleNamespace]:
    """The processes of the machine, as seen by pid 10, a child of pid 9."""
    found: list[SimpleNamespace] = []
    me = SimpleNamespace(pid=10, parents=lambda: [SimpleNamespace(pid=9)])
    monkeypatch.setattr(psutil, "Process", lambda: me)
    monkeypatch.setattr(psutil, "process_iter", lambda attrs: found)
    return found


class TestAlreadyRunning:
    """Tests for finding another copy of the agent."""

    def test_should_find_another_python_that_runs_the_script(
        self, processes: list[SimpleNamespace]
    ) -> None:
        processes.append(process(20, "python3.12", "python", "/opt/agent.py"))
        assert already_running()

    def test_should_ignore_itself_and_its_ancestors(
        self, processes: list[SimpleNamespace]
    ) -> None:
        processes.append(process(10, "python3.12", "python", "/opt/agent.py"))
        processes.append(process(9, "python3.12", "python", "/opt/agent.py"))
        assert not already_running()

    def test_should_ignore_other_processes(
        self, processes: list[SimpleNamespace]
    ) -> None:
        processes.append(process(20, "uv", "uv", "run", "/opt/agent.py"))
        processes.append(process(21, "python3.12", "python", "/opt/controller.py"))
        processes.append(process(22, None))
        assert not already_running()


class Process:
    """Stands in for psutil.Process: records each priority it is given, or refuses."""

    def __init__(self, *, refusing: bool = False) -> None:
        self.calls: list[tuple[str, int]] = []
        self.refusing = refusing

    def nice(self, value: int) -> None:
        if self.refusing:
            raise PermissionError("not allowed")
        self.calls.append(("nice", value))

    def ionice(self, value: int) -> None:
        self.calls.append(("ionice", value))


def machine(
    monkeypatch: pytest.MonkeyPatch, platform: str, *, refusing: bool = False
) -> Process:
    p = Process(refusing=refusing)
    monkeypatch.setattr(agent, "sys", SimpleNamespace(platform=platform))
    monkeypatch.setattr(psutil, "Process", lambda: p)
    monkeypatch.setattr(psutil, "IDLE_PRIORITY_CLASS", 64, raising=False)
    monkeypatch.setattr(psutil, "IOPRIO_VERYLOW", 0, raising=False)
    monkeypatch.setattr(psutil, "IOPRIO_CLASS_IDLE", 3, raising=False)
    return p


class TestDeprioritize:
    """Tests for giving the agent the lowest CPU and IO priority."""

    def test_should_use_the_idle_classes_on_windows(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        p = machine(monkeypatch, "win32")
        deprioritize()
        assert p.calls == [("nice", 64), ("ionice", 0)]

    def test_should_use_the_lowest_niceness_and_idle_io_on_linux(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        p = machine(monkeypatch, "linux")
        deprioritize()
        assert p.calls == [("nice", 19), ("ionice", 3)]

    def test_should_set_only_the_niceness_on_other_systems(
        self, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        p = machine(monkeypatch, "darwin")
        deprioritize()
        assert p.calls == [("nice", 19)]

    def test_should_carry_on_when_the_system_refuses(
        self, monkeypatch: pytest.MonkeyPatch, capsys: pytest.CaptureFixture[str]
    ) -> None:
        p = machine(monkeypatch, "linux", refusing=True)
        deprioritize()
        assert p.calls == []
        assert capsys.readouterr().out == (
            "deprioritize failed: PermissionError: not allowed\n"
        )


class TestConnect:
    """Tests for logging in to the Core and finding the channels of the runner."""

    def test_should_log_in_with_the_api_key_from_the_store(
        self, monkeypatch: pytest.MonkeyPatch, store: Store
    ) -> None:
        client = Core()
        made = cores(monkeypatch, client)
        assert connect(arguments(), NAMES, None, store) is client
        assert store.requests == [(API_KEY_PARAMETER, True)]
        assert made.calls == [
            {
                "host": "core.example",
                "port": 9090,
                "api_key": "syk_test",
                "secure": False,
                "ca_cert": None,
            }
        ]

    def test_should_verify_the_core_with_the_certificate_beside_the_script(
        self, monkeypatch: pytest.MonkeyPatch, store: Store, tmp_path: Path
    ) -> None:
        monkeypatch.setattr(agent, "CA_CERT", tmp_path / "ca.crt")
        made = cores(monkeypatch, Core(), Core())
        connect(arguments(secure=True), NAMES, None, store)
        agent.CA_CERT.write_text("certificate")
        connect(arguments(secure=True), NAMES, None, store)
        assert [c["ca_cert"] for c in made.calls] == [None, str(agent.CA_CERT)]

    def test_should_wait_until_the_controller_creates_the_channels(
        self,
        monkeypatch: pytest.MonkeyPatch,
        store: Store,
        sleeps: list[sy.TimeSpan],
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        early, ready = Core(USAGE), Core()
        cores(monkeypatch, early, ready)
        assert connect(arguments(), NAMES, None, store) is ready
        assert early.closed
        assert sleeps == [RETRY]
        assert capsys.readouterr().out.startswith(
            "startup failed: LookupError: channels not created yet"
        )

    def test_should_need_only_the_usage_channels_without_a_runner(
        self, monkeypatch: pytest.MonkeyPatch, store: Store
    ) -> None:
        client = Core(USAGE)
        cores(monkeypatch, client)
        assert connect(arguments(no_runner=True), NAMES, None, store) is client

    def test_should_read_the_key_again_after_each_failure(
        self,
        monkeypatch: pytest.MonkeyPatch,
        store: Store,
        sleeps: list[sy.TimeSpan],
    ) -> None:
        store.errors = 1
        made = cores(monkeypatch, ConnectionError("Core unreachable"), Core())
        connect(arguments(), NAMES, None, store)
        assert len(store.requests) == 3
        assert len(made.calls) == 2
        assert sleeps == [RETRY, RETRY]

    def test_should_start_the_runner_when_the_core_stays_away(
        self,
        monkeypatch: pytest.MonkeyPatch,
        store: Store,
        sleeps: list[sy.TimeSpan],
    ) -> None:
        monkeypatch.setattr(agent, "CONNECT_GRACE", sy.TimeSpan(0))
        cores(monkeypatch, ConnectionError("Core unreachable"), Core())
        c = Control()
        connect(arguments(), NAMES, control(c), store)
        assert c.applied == [(True, "no Synnax")]

    def test_should_leave_a_runner_that_has_a_state(
        self, monkeypatch: pytest.MonkeyPatch, store: Store
    ) -> None:
        monkeypatch.setattr(agent, "CONNECT_GRACE", sy.TimeSpan(0))
        cores(monkeypatch, Core())
        c = Control()
        c.state = False
        connect(arguments(), NAMES, control(c), store)
        assert c.applied == []


@pytest.fixture
def session(
    monkeypatch: pytest.MonkeyPatch, tmp_path: Path, sleeps: list[sy.TimeSpan]
) -> Core:
    """The Core of a session that runs two passes and finds no runner page logs."""
    client = Core()
    cores(monkeypatch, client)
    monkeypatch.setattr(sy, "Loop", lambda interval: Loop(2))
    monkeypatch.setattr(agent_log, "LOG_DIRS", (tmp_path,))
    return client


class TestRun:
    """Tests for one connected session of the agent."""

    def test_should_write_a_sample_of_each_usage_channel_on_each_pass(
        self, session: Core, store: Store
    ) -> None:
        run(arguments(no_runner=True), NAMES, RUNNER, None, store)
        assert [sorted(f) for f in session.frames] == [sorted(USAGE)] * 2
        first, second = session.written(NAMES["index"])
        assert cast(sy.TimeStamp, first) < cast(sy.TimeStamp, second)
        assert all(
            isinstance(f[NAMES[k]], np.float32)
            for f in session.frames
            for k in ("cpu", "mem", "disk", "disk_percent")
        )
        assert session.closed

    def test_should_drive_the_runner_control_and_the_log(
        self, session: Core, store: Store
    ) -> None:
        c = Control()
        run(arguments(), NAMES, RUNNER, control(c), store)
        assert c.listened == [(NAMES["cmd"], NAMES["state"])]
        assert c.ticks == 2
        assert not c.listening
        assert session.written(NAMES["log"]) == [["[agent] agent connected"]]

    def test_should_open_a_new_writer_after_a_failed_write(
        self, session: Core, store: Store, sleeps: list[sy.TimeSpan]
    ) -> None:
        session.rejected = {NAMES["cpu"]}
        run(arguments(no_runner=True), NAMES, RUNNER, None, store)
        assert session.writers == 2
        assert len(session.written(NAMES["cpu"])) == 1
        assert sleeps == [REOPEN_DELAY]

    def test_should_end_the_session_after_three_failed_writes_in_a_row(
        self,
        session: Core,
        store: Store,
        sleeps: list[sy.TimeSpan],
        monkeypatch: pytest.MonkeyPatch,
    ) -> None:
        monkeypatch.setattr(sy, "Loop", lambda interval: Loop(5))
        session.failing = True
        with pytest.raises(ConnectionError):
            run(arguments(no_runner=True), NAMES, RUNNER, None, store)
        assert session.writers == 3
        assert sleeps == [REOPEN_DELAY, REOPEN_DELAY]
        assert session.closed

    def test_should_open_a_new_log_that_keeps_the_lines_of_a_failed_write(
        self, session: Core, store: Store
    ) -> None:
        session.rejected = {NAMES["log"]}
        c = Control()
        run(arguments(), NAMES, RUNNER, control(c), store)
        assert session.written(NAMES["log"]) == [["[agent] agent connected"]]
        assert isinstance(c.sink, Log)
        assert c.sink is not c.sinks[0]


class Sessions:
    """Stands in for run: records each session, fails the first, and stops the agent on
    the second."""

    def __init__(self) -> None:
        self.calls: list[tuple] = []

    def __call__(self, *args: object) -> None:
        self.calls.append(args)
        if len(self.calls) == 1:
            raise RuntimeError("lost")
        raise KeyboardInterrupt


@pytest.fixture
def sessions(
    monkeypatch: pytest.MonkeyPatch, store: Store, sleeps: list[sy.TimeSpan]
) -> Sessions:
    """An agent whose machine has no other copy and whose sessions are recorded."""
    s = Sessions()
    monkeypatch.setattr(agent, "run", s)
    monkeypatch.setattr(agent, "already_running", lambda: False)
    monkeypatch.setattr(agent, "deprioritize", lambda: None)
    monkeypatch.setattr(agent, "runner_name", lambda: RUNNER)
    monkeypatch.setattr(agent.boto3, "client", lambda *args, **kwargs: store)
    return s


def start(monkeypatch: pytest.MonkeyPatch, *argv: str) -> None:
    monkeypatch.setattr(sys, "argv", ["agent.py", *argv])
    agent.main()


class TestMain:
    """Tests for the start of the agent."""

    def test_should_run_sessions_until_it_is_stopped(
        self,
        sessions: Sessions,
        store: Store,
        sleeps: list[sy.TimeSpan],
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        with pytest.raises(KeyboardInterrupt):
            start(monkeypatch, "--host", "core.example", "--secure")
        args, names, runner, c, used = sessions.calls[1]
        assert (args.host, args.port, args.secure) == ("core.example", 9090, True)
        assert names == NAMES
        assert runner == RUNNER
        assert isinstance(c, Runner)
        assert used is store
        assert sleeps == [SESSION_DELAY]
        assert capsys.readouterr().out == "session failed: RuntimeError: lost\n"

    def test_should_report_only_usage_under_a_given_name(
        self, sessions: Sessions, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        with pytest.raises(KeyboardInterrupt):
            start(monkeypatch, "--host", "h", "--runner-name", "sandbox", "--no-runner")
        _, names, runner, c, _ = sessions.calls[0]
        assert names["cpu"] == "sandbox_cpu"
        assert runner == "sandbox"
        assert c is None

    def test_should_exit_when_another_copy_runs(
        self,
        sessions: Sessions,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setattr(agent, "already_running", lambda: True)
        start(monkeypatch, "--host", "core.example")
        assert sessions.calls == []
        assert capsys.readouterr().out == "another copy is running; exiting\n"

    def test_should_exit_on_a_machine_that_is_not_a_runner(
        self,
        sessions: Sessions,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        monkeypatch.setattr(agent, "runner_name", lambda: None)
        start(monkeypatch, "--host", "core.example")
        assert sessions.calls == []
        assert capsys.readouterr().out == (
            "no .runner file after waiting; not a runner\n"
        )

    def test_should_need_the_host_of_the_core(
        self, sessions: Sessions, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        with pytest.raises(SystemExit):
            start(monkeypatch)
