#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import os
import stat
import sys
from collections.abc import Iterator
from pathlib import Path

import pytest

from x.fs import stream_to_file


@pytest.fixture
def umask() -> Iterator[None]:
    previous = os.umask(0o027)
    yield
    os.umask(previous)


def _mode(path: Path) -> int:
    return stat.S_IMODE(path.stat().st_mode)


class TestStreamToFile:
    def test_writes_chunks_in_order(self, tmp_path: Path) -> None:
        """Should write the concatenated chunks and leave no temp file behind."""
        dest = tmp_path / "out.json"
        stream_to_file([b'{"a":', b" 1}"], dest)
        assert dest.read_bytes() == b'{"a": 1}'
        assert list(tmp_path.glob("*.part")) == []

    def test_overwrites_existing_dest(self, tmp_path: Path) -> None:
        """Should atomically replace an existing destination file."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')
        stream_to_file([b'{"new": true}'], dest)
        assert dest.read_bytes() == b'{"new": true}'

    def test_empty_stream_allowed_by_default(self, tmp_path: Path) -> None:
        """An empty stream writes a zero-byte file when allow_empty is left at
        default."""
        dest = tmp_path / "out.json"
        stream_to_file([], dest)
        assert dest.read_bytes() == b""
        assert list(tmp_path.glob("*.part")) == []

    def test_empty_stream_rejected_when_disallowed(self, tmp_path: Path) -> None:
        """An empty stream raises and writes nothing when allow_empty is False."""
        dest = tmp_path / "out.json"
        with pytest.raises(ValueError, match="empty stream"):
            stream_to_file([], dest, allow_empty=False)
        assert not dest.exists()
        assert list(tmp_path.glob("*.part")) == []

    def test_empty_stream_disallowed_preserves_existing_dest(
        self, tmp_path: Path
    ) -> None:
        """Rejecting an empty stream leaves an existing dest untouched."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')
        with pytest.raises(ValueError, match="empty stream"):
            stream_to_file([], dest, allow_empty=False)
        assert dest.read_bytes() == b'{"old": true}'
        assert list(tmp_path.glob("*.part")) == []

    def test_mid_stream_failure_preserves_dest(self, tmp_path: Path) -> None:
        """A failure partway through leaves an existing dest untouched and no temp
        file."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')

        def chunks() -> Iterator[bytes]:
            yield b'{"partial'
            raise ConnectionError("stream broke")

        with pytest.raises(ConnectionError, match="stream broke"):
            stream_to_file(chunks(), dest)
        assert dest.read_bytes() == b'{"old": true}'
        assert list(tmp_path.glob("*.part")) == []

    @pytest.mark.skipif(sys.platform == "win32", reason="POSIX file modes")
    @pytest.mark.usefixtures("umask")
    def test_new_dest_mode_follows_umask(self, tmp_path: Path) -> None:
        """Should create dest with the mode that open() gives a new file."""
        dest = tmp_path / "out.json"
        stream_to_file([b"{}"], dest)
        expected = tmp_path / "expected.json"
        expected.write_bytes(b"{}")
        assert _mode(dest) == _mode(expected) == 0o640

    @pytest.mark.skipif(sys.platform == "win32", reason="POSIX file modes")
    def test_keeps_existing_dest_mode(self, tmp_path: Path) -> None:
        """Should keep the mode of the dest that it replaces."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')
        dest.chmod(0o604)
        stream_to_file([b'{"new": true}'], dest)
        assert _mode(dest) == 0o604

    @pytest.mark.skipif(sys.platform == "win32", reason="POSIX file modes")
    @pytest.mark.usefixtures("umask")
    def test_replaced_dest_streams_privately(self, tmp_path: Path) -> None:
        """Should not let other users read the stream that replaces a private dest."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b"{}")
        dest.chmod(0o600)
        modes: list[int] = []

        def chunks() -> Iterator[bytes]:
            modes.extend(_mode(p) for p in tmp_path.glob("*.part"))
            yield b"{}"

        stream_to_file(chunks(), dest)
        assert modes == [0o600]
        assert _mode(dest) == 0o600

    def test_empty_stream_over_read_only_dest(self, tmp_path: Path) -> None:
        """Should leave a read-only dest and no temp file after an empty stream."""
        dest = tmp_path / "out.json"
        dest.write_bytes(b'{"old": true}')
        dest.chmod(stat.S_IREAD)
        with pytest.raises(ValueError, match="empty stream"):
            stream_to_file([], dest, allow_empty=False)
        assert dest.read_bytes() == b'{"old": true}'
        assert list(tmp_path.glob("*.part")) == []

    def test_long_dest_name(self, tmp_path: Path) -> None:
        """Should write a dest whose name is at the file name length limit."""
        dest = tmp_path / ("a" * 250 + ".json")
        stream_to_file([b"{}"], dest)
        assert dest.read_bytes() == b"{}"
