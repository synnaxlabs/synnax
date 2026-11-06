#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]


def run(script: str) -> str:
    return subprocess.run(
        ["uv", "run", "--project", "client/py", "python", f"scripts/{script}"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        check=True,
    ).stdout


class TestCLIReport:
    def test_lists_commands_and_their_options_without_help(self) -> None:
        lines = run("surface_py_cli.py").splitlines()
        assert "sy command" in lines
        assert "sy login command" in lines
        assert any(line.startswith("sy tsconvert input_path {") for line in lines)
        assert not any('"help"' in line for line in lines)

    def test_reports_the_same_text_on_each_run(self) -> None:
        assert run("surface_py_cli.py") == run("surface_py_cli.py")


class TestStateReport:
    def test_lists_the_config_keys_and_keyring_entries(self) -> None:
        assert run("surface_py_state.py").splitlines() == [
            "config.json.clusters.cluster.options.ca_cert str",
            "config.json.clusters.cluster.options.host str",
            "config.json.clusters.cluster.options.port int",
            "config.json.clusters.cluster.options.secure bool",
            "keyring synnax.api_key cluster@host:9090",
        ]
