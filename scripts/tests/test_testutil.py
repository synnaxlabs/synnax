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

from scripts_common import FakeGH


def gh_exit_code(gh: FakeGH, *args: str) -> int:
    return subprocess.run(
        [str(gh.bin_dir / "gh"), *args], capture_output=True, text=True
    ).returncode


class TestFakeGH:
    def test_keeps_a_consumed_failure_when_another_route_adds_one(
        self, tmp_path: Path
    ) -> None:
        """Should not restore a used failure when a later `fail` adds another route."""
        gh = FakeGH(tmp_path)
        gh.serve("api first")
        gh.serve("api second")
        gh.fail("api first", "boom")
        assert gh_exit_code(gh, "api", "first") == 1
        gh.fail("api second", "boom")
        assert gh_exit_code(gh, "api", "first") == 0
        assert gh_exit_code(gh, "api", "second") == 1
        assert gh_exit_code(gh, "api", "second") == 0
