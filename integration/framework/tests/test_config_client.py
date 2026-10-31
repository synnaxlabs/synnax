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
from collections.abc import Iterator
from pathlib import Path
from typing import Any

import pytest

from framework import config_client
from framework.target_filter import TargetFilter

GENERIC_CASE = """
from typing import Generic, TypeVar

from pydantic import BaseModel

from framework.test_case import TestCase

T = TypeVar("T")


class Box(BaseModel, Generic[T]):
    value: T


IntBox = Box[int]


class GenericCase(TestCase):
    def run(self) -> None:
        pass
"""


def named_case(name: str) -> str:
    return f"""
from framework.test_case import TestCase


class {name}(TestCase):
    def run(self) -> None:
        pass
"""


@pytest.fixture
def cases_dir(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Iterator[Path]:
    monkeypatch.chdir(tmp_path)
    before = set(sys.modules)
    yield tmp_path / "tests"
    for name in set(sys.modules) - before:
        if name.startswith("tests.config_client_spec"):
            del sys.modules[name]


def write_case(cases_dir: Path, *, case: str, source: str) -> None:
    path = cases_dir / f"{case}.py"
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(source)


def load(
    cases_dir: Path, *, sequence: dict[str, Any]
) -> tuple[list[config_client.Sequence], list[config_client.TestDefinition]]:
    (cases_dir / "spec_tests.json").write_text(json.dumps({"sequences": [sequence]}))
    return config_client.ConfigClient(tests_dir=cases_dir).load(
        TargetFilter(file_filter=["spec"])
    )


class TestConfigClient:
    def test_generic_model_at_import(self, cases_dir: Path) -> None:
        """Should load a case that parametrizes a generic Pydantic model."""
        case = "config_client_spec/generic"
        write_case(cases_dir, case=case, source=GENERIC_CASE)
        loaded = config_client.ConfigClient().load_test_class(
            config_client.TestDefinition(case=case)
        )
        assert loaded.__name__ == "GenericCase"

    def test_same_file_name_in_two_directories(self, cases_dir: Path) -> None:
        """Should keep apart cases that share a file name in different directories."""
        write_case(
            cases_dir, case="config_client_spec/a/case", source=named_case("CaseA")
        )
        write_case(
            cases_dir, case="config_client_spec/b/case", source=named_case("CaseB")
        )
        client = config_client.ConfigClient()
        a = client.load_test_class(
            config_client.TestDefinition(case="config_client_spec/a/case")
        )
        b = client.load_test_class(
            config_client.TestDefinition(case="config_client_spec/b/case")
        )
        assert a.__name__ == "CaseA"
        assert b.__name__ == "CaseB"
        assert a.__module__ != b.__module__

    def test_timeout(self, cases_dir: Path) -> None:
        """Should apply the default timeout and keep a per-test timeout through
        parameter expansion."""
        write_case(
            cases_dir, case="config_client_spec/plain", source=named_case("Plain")
        )
        write_case(cases_dir, case="config_client_spec/slow", source=named_case("Slow"))
        sequence = {
            "sequence_name": "timeouts",
            "tests": [
                {"case": "config_client_spec/plain"},
                {
                    "case": "config_client_spec/slow",
                    "timeout": 12.5,
                    "parameters": {"n": [1, 2]},
                },
            ],
        }
        _, defs = load(cases_dir, sequence=sequence)
        timeouts = {str(d): d.timeout for d in defs}
        assert timeouts == {
            "config_client_spec/plain": config_client._DEFAULT_TIMEOUT,
            "config_client_spec/slow (slow_1)": 12.5,
            "config_client_spec/slow (slow_2)": 12.5,
        }

    @pytest.mark.parametrize("timeout", [None, 0, -1, "10", True, float("inf")])
    def test_invalid_timeout(self, cases_dir: Path, timeout: Any) -> None:
        """Should reject a timeout that is not a finite, positive number."""
        write_case(
            cases_dir, case="config_client_spec/plain", source=named_case("Plain")
        )
        sequence = {
            "sequence_name": "timeouts",
            "tests": [{"case": "config_client_spec/plain", "timeout": timeout}],
        }
        with pytest.raises(ValueError, match="timeout"):
            load(cases_dir, sequence=sequence)

    def test_sequence_not_an_object(self, cases_dir: Path) -> None:
        """Should fail on a sequence entry that is not an object."""
        cases_dir.mkdir(parents=True)
        sequences = [{"sequence_name": "ok", "tests": []}, "bad"]
        (cases_dir / "spec_tests.json").write_text(json.dumps({"sequences": sequences}))
        with pytest.raises(
            ValueError,
            match=r"^Sequence 1 in test file 'spec_tests\.json' is not an object$",
        ):
            config_client.ConfigClient(tests_dir=cases_dir).load(
                TargetFilter(file_filter=["spec"])
            )
