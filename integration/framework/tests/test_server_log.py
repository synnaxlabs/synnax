#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path

import pytest

from framework import server_log


class TestServerLog:
    """The server_log functions read only the log path they receive."""

    def test_should_ignore_the_environment(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        other = tmp_path / "other.log"
        other.write_bytes(b"other")
        monkeypatch.setenv("SYNNAX_SERVER_LOG", str(other))
        assert server_log.current_offset(tmp_path / "missing.log") is None

    def test_should_slice_the_given_log(self, tmp_path: Path) -> None:
        src = tmp_path / "core.log"
        src.write_bytes(b"before|during")
        start = len(b"before|")
        dst = tmp_path / "bundle" / "server.log"
        assert server_log.slice_to(
            src, dst, start=start, end=server_log.current_offset(src)
        )
        assert dst.read_bytes() == b"during"

    def test_should_copy_from_start_to_the_end(self, tmp_path: Path) -> None:
        src = tmp_path / "core.log"
        src.write_bytes(b"old")
        start = server_log.current_offset(src)
        with open(src, "ab") as f:
            f.write(b"new")
        dst = tmp_path / "server.log"
        assert server_log.copy_range(src, dst, start=start)
        assert dst.read_bytes() == b"new"

    def test_should_skip_a_missing_log(self, tmp_path: Path) -> None:
        dst = tmp_path / "server.log"
        assert not server_log.copy_range(tmp_path / "missing.log", dst, start=0)
        assert not dst.exists()
