#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

from framework.models import DEFAULT_OPTIONS
from framework.test_conductor import parse_args, run_options


class TestRunOptions:
    def test_binaries(self) -> None:
        """Should hand the Core and Driver binaries to test cases."""
        args = parse_args(["--core-binary", "core", "--driver-binary", "driver"])
        options = run_options(args)
        assert options.core_binary == Path("core")
        assert options.driver_binary == Path("driver")

    def test_defaults(self) -> None:
        """Should match the options of a run with no flags."""
        assert run_options(parse_args([])) == DEFAULT_OPTIONS
