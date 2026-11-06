#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import sys
import threading
from collections.abc import Callable, Iterator
from pathlib import Path
from typing import Any, cast

import pytest
from pydantic import BaseModel

import synnax as sy
from framework import models, resources
from framework.config_client import ConfigClient, Sequence
from framework.cores import Core
from framework.execution_client import ExecutionClient
from framework.models import DEFAULT_OPTIONS, STATUS, SynnaxConnection
from framework.resources import Resources
from framework.target_filter import TargetFilter

FIRST_PORT = 9090

# A case that records when and on which Core it ran. It skips the TestCase
# constructor, which connects to a Core.
HOLD_CASE = """
import json
import time

from framework import resources
from framework.log_client import LogClient
from framework.models import STATUS
from framework.test_case import TestCase


class Hold(TestCase):
    @classmethod
    def resources(cls) -> frozenset[str]:
        return super().resources() | {HELD}

    def __init__(self, *, synnax_connection, name, options, results_dir, params):
        self.synnax_connection = synnax_connection
        self.name = name
        self.params = params
        self._status = STATUS.INITIALIZING
        self._error_message = None
        self._error_traceback = None
        self.skip_reason = None
        self.log_client = LogClient(name=name)

    def run(self) -> None:
        pass

    def execute(self) -> None:
        start = time.monotonic()
        time.sleep(self.params["hold"])
        span = {
            "name": self.name,
            "port": self.synnax_connection.port,
            "start": start,
            "end": time.monotonic(),
        }
        with open(self.params["record"], "a") as f:
            f.write(json.dumps(span) + "\\n")
        self._status = STATUS.PASSED
"""

CASES = {
    "plain": "",
    "ni": "resources.NI",
    "sim": 'resources.simulator("sim")',
    "browser": "resources.BROWSER",
}


class Span(BaseModel):
    name: str
    port: int
    start: float
    end: float

    def overlaps(self, other: "Span") -> bool:
        return self.start < other.end and other.start < self.end


@pytest.fixture
def cases_dir(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Iterator[Path]:
    monkeypatch.chdir(tmp_path)
    before = set(sys.modules)
    cases = tmp_path / "tests" / "lanes_spec"
    cases.mkdir(parents=True)
    for name, held in CASES.items():
        source = HOLD_CASE.replace("{HELD}", "{" + held + "}" if held else "set()")
        (cases / f"{name}.py").write_text(source)
    yield tmp_path / "tests"
    for name in set(sys.modules) - before:
        if name.startswith("tests.lanes_spec"):
            del sys.modules[name]


def sequence(
    name: str,
    *cases: str,
    hold: float = 0.3,
    order: str = "sequential",
    isolated: bool = False,
    timeout: float | None = None,
) -> dict[str, object]:
    return {
        "sequence_name": name,
        "sequence_order": order,
        "isolated": isolated,
        "tests": [
            {
                "case": f"lanes_spec/{case}",
                "name": f"{name}_{i}",
                "parameters": {"hold": hold, "record": "record.jsonl"},
                **({} if timeout is None else {"timeout": timeout}),
            }
            for i, case in enumerate(cases)
        ],
    }


def client(
    cases_dir: Path,
    *sequences: dict[str, object],
    cores: int = 2,
    browsers: int | None = None,
    active: list[tuple[Any, ...]] | None = None,
    on_test_ran: Callable[[], None] = lambda: None,
    timeout_grace: float = 60.0,
) -> tuple[ExecutionClient, list[Sequence], list[models.Test]]:
    """Returns a client for the sequences, with the sequences to run and the list it
    records results in."""
    (cases_dir / "lanes_tests.json").write_text(json.dumps({"sequences": sequences}))
    config = ConfigClient(tests_dir=cases_dir)
    loaded, _ = config.load(TargetFilter(file_filter=["lanes"]))
    results: list[models.Test] = []
    execution = ExecutionClient(
        config_client=config,
        cores=[
            Core(
                connection=SynnaxConnection(port=FIRST_PORT + i),
                log_path=cases_dir / f"core-{i}.log",
            )
            for i in range(cores)
        ],
        resources=Resources({resources.BROWSER: browsers}),
        options=DEFAULT_OPTIONS,
        run_dir=cases_dir.parent / "run",
        # Only a conductor range needs the client, and these runs have none.
        client=cast(sy.Synnax, None),
        conductor_range=None,
        tests=results,
        tests_lock=threading.Lock(),
        active_tests=active if active is not None else [],
        active_tests_lock=threading.Lock(),
        log=lambda message, /, *, use_name=True: None,
        on_status_change=lambda _: None,
        on_test_ran=on_test_ran,
        timeout_grace=timeout_grace,
    )
    return execution, loaded, results


def spans(cases_dir: Path) -> dict[str, Span]:
    record = cases_dir.parent / "record.jsonl"
    lines = record.read_text().splitlines() if record.exists() else []
    return {s.name: s for s in map(Span.model_validate_json, lines)}


def run(
    cases_dir: Path,
    *sequences: dict[str, object],
    cores: int = 2,
    browsers: int | None = None,
) -> dict[str, Span]:
    execution, loaded, results = client(
        cases_dir, *sequences, cores=cores, browsers=browsers
    )
    execution.run(loaded)
    assert [t.status for t in results] == [STATUS.PASSED] * len(results)
    return spans(cases_dir)


class TestExecutionClient:
    def test_free_resources(self, cases_dir: Path) -> None:
        """Should run tests that hold nothing at the same time."""
        spans = run(cases_dir, sequence("a", "plain", "plain", order="asynchronous"))
        assert spans["a_0"].overlaps(spans["a_1"])

    def test_shared_simulator_in_a_sequence(self, cases_dir: Path) -> None:
        """Should never run two tests of a sequence that hold one simulator at the
        same time."""
        spans = run(cases_dir, sequence("a", "sim", "sim", order="asynchronous"))
        assert not spans["a_0"].overlaps(spans["a_1"])

    def test_no_browser_limit(self, cases_dir: Path) -> None:
        """Should run Console tests at the same time with no browser limit."""
        spans = run(
            cases_dir, sequence("a", "browser", "browser", order="asynchronous")
        )
        assert spans["a_0"].overlaps(spans["a_1"])

    def test_sequences_at_once(self, cases_dir: Path) -> None:
        """Should run two sequences at the same time on different Cores."""
        spans = run(cases_dir, sequence("a", "plain"), sequence("b", "plain"))
        assert spans["a_0"].overlaps(spans["b_0"])
        assert spans["a_0"].port != spans["b_0"].port

    def test_order_in_a_sequence(self, cases_dir: Path) -> None:
        """Should run the tests of a sequence one after another on one Core."""
        spans = run(cases_dir, sequence("a", "plain", "plain", hold=0.1))
        assert spans["a_0"].end <= spans["a_1"].start
        assert spans["a_0"].port == spans["a_1"].port

    def test_one_core(self, cases_dir: Path) -> None:
        """Should run sequences in order with one Core."""
        spans = run(cases_dir, sequence("a", "plain"), sequence("b", "plain"), cores=1)
        assert spans["a_0"].end <= spans["b_0"].start

    def test_ni_on_first_core(self, cases_dir: Path) -> None:
        """Should run a sequence that holds NI on the first Core only."""
        spans = run(
            cases_dir,
            sequence("a", "plain"),
            sequence("b", "plain"),
            sequence("c", "ni"),
            cores=3,
        )
        assert spans["c_0"].port == FIRST_PORT

    def test_shared_simulator(self, cases_dir: Path) -> None:
        """Should never run two tests that hold one simulator at the same time."""
        spans = run(cases_dir, sequence("a", "sim"), sequence("b", "sim"))
        assert not spans["a_0"].overlaps(spans["b_0"])

    def test_browser_limit(self, cases_dir: Path) -> None:
        """Should run no more Console tests at once than the browser limit."""
        spans = run(
            cases_dir, sequence("a", "browser"), sequence("b", "browser"), browsers=1
        )
        assert not spans["a_0"].overlaps(spans["b_0"])

    def test_isolated_last(self, cases_dir: Path) -> None:
        """Should run an isolated sequence alone on the first Core after the
        others."""
        spans = run(
            cases_dir,
            sequence("t", "plain", isolated=True),
            sequence("a", "plain"),
            sequence("b", "plain"),
        )
        assert spans["t_0"].start >= max(spans["a_0"].end, spans["b_0"].end)
        assert spans["t_0"].port == FIRST_PORT

    def test_isolated_after_abandoned(self, cases_dir: Path) -> None:
        """Should fail an isolated sequence while an abandoned test still runs."""
        before = set(threading.enumerate())
        execution, loaded, results = client(
            cases_dir,
            sequence("a", "plain", hold=2, timeout=0.1),
            sequence("t", "plain", isolated=True),
            timeout_grace=0,
        )
        execution.run(loaded)
        for t in set(threading.enumerate()) - before:
            t.join(timeout=5)
        timed_out = "Test exceeded its timeout (0.1s) and did not stop"
        blocked = "Abandoned tests still run: lanes_spec/plain (a_0)"
        assert sorted((t.name, t.status, t.error_message) for t in results) == [
            ("a_0", STATUS.TIMEOUT, timed_out),
            ("t_0", STATUS.FAILED, blocked),
        ]
        assert "t_0" not in spans(cases_dir)

    # A stop ends the running test's thread with SystemExit.
    @pytest.mark.filterwarnings("ignore::pytest.PytestUnhandledThreadExceptionWarning")
    def test_stop_waiting(self, cases_dir: Path) -> None:
        """Should not start a test that waits for a resource when the run stops."""
        active: list[tuple[Any, ...]] = []
        execution, loaded, results = client(
            cases_dir,
            sequence("a", "sim", "sim", hold=1, order="asynchronous"),
            active=active,
        )
        runner = threading.Thread(target=execution.run, args=(loaded,))
        runner.start()
        timer = sy.Timer()
        while not active:
            assert timer.elapsed() < 5 * sy.TimeSpan.SECOND
            sy.sleep(0.01)
        execution.stop()
        runner.join(timeout=10)
        assert not runner.is_alive()
        assert spans(cases_dir) == {}
        assert sorted((t.status, t.error_message or "") for t in results) == [
            (STATUS.KILLED, "Test was manually killed"),
            (STATUS.KILLED, "The run stopped before the test started"),
        ]

    def test_lane_error(self, cases_dir: Path) -> None:
        """Should raise an error from a lane after the lanes end."""

        def on_test_ran() -> None:
            raise RuntimeError("telemetry is down")

        execution, loaded, _ = client(
            cases_dir, sequence("a", "plain"), on_test_ran=on_test_ran
        )
        with pytest.raises(RuntimeError, match=r"^telemetry is down$"):
            execution.run(loaded)
