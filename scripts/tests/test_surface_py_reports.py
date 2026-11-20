#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import argparse

import surface_py_cli
import surface_py_state


class TestCLIReport:
    def test_lists_commands_and_their_arguments_without_help(self) -> None:
        lines = surface_py_cli.report().splitlines()
        assert "sy command" in lines
        assert "sy login command" in lines
        assert not any('"help"' in line for line in lines)

    def test_reports_whether_a_command_is_required(self) -> None:
        lines = surface_py_cli.report().splitlines()
        assert 'sy command {"required": true}' in lines

    def test_gives_each_positional_argument_its_position(self) -> None:
        parser = argparse.ArgumentParser(prog="sy")
        parser.add_argument("a")
        parser.add_argument("--flag")
        parser.add_argument("b")
        out: list[str] = []
        surface_py_cli.walk(parser, path="sy", out=out)
        assert '"position": 0' in out[1]
        assert "position" not in out[2]
        assert '"position": 1' in out[3]

    def test_reports_the_same_text_on_each_run(self) -> None:
        assert surface_py_cli.report() == surface_py_cli.report()


class TestStateReport:
    def test_lists_the_config_keys_and_keyring_entries(self) -> None:
        assert surface_py_state.report().splitlines() == [
            "config.json.clusters.default.options.ca_cert str",
            "config.json.clusters.default.options.host str",
            "config.json.clusters.default.options.port int",
            "config.json.clusters.default.options.secure bool",
            "keyring synnax.api_key default@host:9090",
        ]
