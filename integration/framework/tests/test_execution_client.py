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
from collections.abc import Iterator
from pathlib import Path
from typing import cast

import pytest
from pydantic import BaseModel

import synnax as sy
from framework import models, resources
from framework.config_client import ConfigClient
from framework.execution_client import ExecutionClient
from framework.models import DEFAULT_OPTIONS, STATUS, SynnaxConnection
from framework.resources import Resources
from framework.target_filter import TargetFilter

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


def run(cases_dir: Path, *cases: str, browsers: int | None = None) -> dict[str, Span]:
    """Runs the cases as one asynchronous sequence, so each starts at once."""
    sequence = {
        "sequence_name": "a",
        "sequence_order": "asynchronous",
        "tests": [
            {
                "case": f"lanes_spec/{case}",
                "name": f"a_{i}",
                "parameters": {"hold": 0.3, "record": "record.jsonl"},
            }
            for i, case in enumerate(cases)
        ],
    }
    (cases_dir / "lanes_tests.json").write_text(json.dumps({"sequences": [sequence]}))
    config = ConfigClient(tests_dir=cases_dir)
    loaded, _ = config.load(TargetFilter(file_filter=["lanes"]))
    results: list[models.Test] = []
    client = ExecutionClient(
        config_client=config,
        synnax_connection=SynnaxConnection(),
        resources=Resources({resources.BROWSER: browsers}),
        options=DEFAULT_OPTIONS,
        server_log_path=cases_dir / "core.log",
        run_dir=cases_dir.parent / "run",
        # Only a conductor range needs the client, and these runs have none.
        client=cast(sy.Synnax, None),
        conductor_range=None,
        tests=results,
        tests_lock=threading.Lock(),
        active_tests=[],
        active_tests_lock=threading.Lock(),
        log=lambda message, /, *, use_name=True: None,
        on_status_change=lambda _: None,
        on_test_ran=lambda: None,
    )
    client.run(loaded)
    assert [t.status for t in results] == [STATUS.PASSED] * len(results)
    lines = (cases_dir.parent / "record.jsonl").read_text().splitlines()
    spans = [Span.model_validate_json(line) for line in lines]
    return {s.name: s for s in spans}


class TestExecutionClient:
    def test_free_resources(self, cases_dir: Path) -> None:
        """Should run tests that hold nothing at the same time."""
        spans = run(cases_dir, "plain", "plain")
        assert spans["a_0"].overlaps(spans["a_1"])

    def test_shared_simulator(self, cases_dir: Path) -> None:
        """Should never run two tests that hold one simulator at the same time."""
        spans = run(cases_dir, "sim", "sim")
        assert not spans["a_0"].overlaps(spans["a_1"])

    def test_browser_limit(self, cases_dir: Path) -> None:
        """Should run no more Console tests at once than the browser limit."""
        spans = run(cases_dir, "browser", "browser", browsers=1)
        assert not spans["a_0"].overlaps(spans["a_1"])

    def test_no_browser_limit(self, cases_dir: Path) -> None:
        """Should run Console tests at the same time with no browser limit."""
        spans = run(cases_dir, "browser", "browser")
        assert spans["a_0"].overlaps(spans["a_1"])
