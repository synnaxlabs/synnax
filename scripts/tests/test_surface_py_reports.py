#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import surface_py_cli
import surface_py_state


class TestCLIReport:
    def test_lists_commands_and_their_arguments_without_help(self) -> None:
        lines = surface_py_cli.report().splitlines()
        assert "sy command" in lines
        assert "sy login command" in lines
        assert any(line.startswith("sy tsconvert input_path {") for line in lines)
        assert not any('"help"' in line for line in lines)

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
