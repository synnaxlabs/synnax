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
from typing import Any

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
        self.metrics = {}
        self.log_client = LogClient(name=name)

    def run(self) -> None:
        pass

    def execute(self) -> None:
        self.metrics["hold_s"] = self.params["hold"]
        start = time.monotonic()
        time.sleep(self.params["hold"])
        span = {
            "name": self.name,
            "port": self.synnax_connection.port,
            "start": start,
            "end": time.monotonic(),
        }
        with open(self.params["record"], "a+") as f:
            f.write(json.dumps(span) + "\\n")
            f.seek(0)
            attempts = sum(json.loads(line)["name"] == self.name for line in f)
        failed = attempts == 1 and self.name == self.params["fails_once"]
        self._status = STATUS.FAILED if failed else STATUS.PASSED
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
    independent: bool = False,
    timeout: float | None = None,
    fails_once: str = "",
) -> dict[str, object]:
    """:param fails_once: Name of the test that fails the first time it runs."""
    return {
        "sequence_name": name,
        "sequence_order": order,
        "isolated": isolated,
        "independent": independent,
        "tests": [
            {
                "case": f"lanes_spec/{case}",
                "name": f"{name}_{i}",
                "parameters": {
                    "hold": hold,
                    "record": "record.jsonl",
                    "fails_once": fails_once,
                },
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
    log: Callable[[str], None] = lambda message: None,
    conductor_range: sy.Range | None = None,
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
        client=None,
        conductor_range=conductor_range,
        tests=results,
        tests_lock=threading.Lock(),
        active_tests=active if active is not None else [],
        active_tests_lock=threading.Lock(),
        log=lambda message, /, *, use_name=True: log(message),
        on_status_change=lambda _: None,
        on_test_ran=on_test_ran,
        timeout_grace=timeout_grace,
    )
    return execution, loaded, results


def attempts(cases_dir: Path) -> dict[str, list[Span]]:
    """Returns the spans of each test by name, one for each time the test ran."""
    record = cases_dir.parent / "record.jsonl"
    lines = record.read_text().splitlines() if record.exists() else []
    by_name: dict[str, list[Span]] = {}
    for span in map(Span.model_validate_json, lines):
        by_name.setdefault(span.name, []).append(span)
    return by_name


def spans(cases_dir: Path) -> dict[str, Span]:
    return {name: ran[-1] for name, ran in attempts(cases_dir).items()}


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
    def test_metrics(self, cases_dir: Path) -> None:
        """Should keep the numbers that each test measured in its result."""
        execution, loaded, results = client(cases_dir, sequence("a", "plain", hold=0.1))
        execution.run(loaded)
        assert [t.metrics for t in results] == [{"hold_s": 0.1}]

    def test_metrics_of_timeout(self, cases_dir: Path) -> None:
        """Should keep the numbers that a test measured before it timed out."""
        execution, loaded, results = client(
            cases_dir, sequence("a", "plain", hold=2, timeout=0.1)
        )
        execution.run(loaded)
        assert [(t.status, t.metrics) for t in results] == [
            (STATUS.TIMEOUT, {"hold_s": 2})
        ]

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

    def test_conductor_range_without_client(self, cases_dir: Path) -> None:
        """Should reject a conductor range that comes without a client."""
        conductor_range = sy.Range(
            name="conductor",
            time_range=sy.TimeStamp.now().span_range(sy.TimeSpan.SECOND),
        )
        with pytest.raises(ValueError, match="needs a client"):
            client(cases_dir, sequence("a", "plain"), conductor_range=conductor_range)

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

    def test_retry_with_peers(self, cases_dir: Path) -> None:
        """Should run the whole asynchronous sequence again for a failed test, and
        keep one result for each test."""
        ran_count = 0

        def on_test_ran() -> None:
            nonlocal ran_count
            ran_count += 1

        execution, loaded, results = client(
            cases_dir,
            sequence("a", "plain", "plain", order="asynchronous", fails_once="a_1"),
            on_test_ran=on_test_ran,
        )
        execution.run(loaded)
        assert ran_count == 4
        ran = attempts(cases_dir)
        assert [len(ran[name]) for name in ("a_0", "a_1")] == [2, 2]
        assert ran["a_0"][1].overlaps(ran["a_1"][1])
        assert sorted((t.name, t.status) for t in results) == [
            ("a_0", STATUS.PASSED),
            ("a_1", STATUS.FLAKY),
        ]

    def test_retry_stopped_before_peers(self, cases_dir: Path) -> None:
        """Should keep the first result of a peer that a stopped retry does not run
        again."""

        def log(message: str) -> None:
            if "RETRYING" in message:
                execution.stop()

        execution, loaded, results = client(
            cases_dir,
            sequence("a", "plain", "plain", order="asynchronous", fails_once="a_1"),
            log=log,
        )
        execution.run(loaded)
        assert len(attempts(cases_dir)["a_0"]) == 1
        assert [(t.name, t.status) for t in results] == [("a_0", STATUS.PASSED)]

    # A stop ends the running test's thread with SystemExit.
    @pytest.mark.filterwarnings("ignore::pytest.PytestUnhandledThreadExceptionWarning")
    def test_retry_stopped_during_peers(self, cases_dir: Path) -> None:
        """Should keep one result for each test when the run stops during a retry, and
        the first result of a peer whose second run did not start."""
        retrying = threading.Event()

        def log(message: str) -> None:
            if "RETRYING" in message:
                retrying.set()

        active: list[tuple[Any, ...]] = []
        execution, loaded, results = client(
            cases_dir,
            sequence(
                "a", "sim", "sim", hold=0.3, order="asynchronous", fails_once="a_1"
            ),
            active=active,
            log=log,
        )
        runner = threading.Thread(target=execution.run, args=(loaded,))
        runner.start()
        timer = sy.Timer()
        while not (retrying.is_set() and active):
            assert timer.elapsed() < 10 * sy.TimeSpan.SECOND
            sy.sleep(0.01)
        execution.stop()
        runner.join(timeout=10)
        assert not runner.is_alive()
        by_name = {t.name: (t.status, t.error_message or "") for t in results}
        assert len(results) == 2
        assert set(by_name) == {"a_0", "a_1"}
        killed = (STATUS.KILLED, "Test was manually killed")
        waiting = (STATUS.KILLED, "The run stopped before the test started")
        assert (by_name["a_0"], by_name["a_1"]) in [
            (killed, waiting),
            ((STATUS.PASSED, ""), killed),
        ]

    @pytest.mark.parametrize(
        ("order", "independent"), [("asynchronous", True), ("sequential", False)]
    )
    def test_retry_without_peers(
        self, cases_dir: Path, order: str, independent: bool
    ) -> None:
        """Should run only the failed test again in an independent sequence and in a
        sequential one."""
        execution, loaded, results = client(
            cases_dir,
            sequence(
                "a",
                "plain",
                "plain",
                order=order,
                independent=independent,
                fails_once="a_1",
            ),
        )
        execution.run(loaded)
        ran = attempts(cases_dir)
        assert [len(ran[name]) for name in ("a_0", "a_1")] == [1, 2]
        assert sorted((t.name, t.status) for t in results) == [
            ("a_0", STATUS.PASSED),
            ("a_1", STATUS.FLAKY),
        ]
