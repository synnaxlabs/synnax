#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import socket
from pathlib import Path

import pytest

import synnax as sy
import synnax.config as config
from freighter.exceptions import Unreachable
from synnax.config import (
    DESKTOP_NOT_RUNNING_MSG,
    NO_OPTIONS_MSG,
    load_desktop_options,
    try_load_options_if_none_provided,
)
from synnax.exceptions import ValidationError
from synnax.options import Options


def write_desktop_file(path: Path, *, port: int, api_key: str = "syk_k_s") -> None:
    """Writes the file the way Synnax Desktop does."""
    path.write_text(
        json.dumps(
            {"host": "127.0.0.1", "port": port, "secure": False, "api_key": api_key}
        )
    )


def closed_port() -> int:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        return int(s.getsockname()[1])


@pytest.fixture
def home(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    """Points the saved logins and the Desktop file at an empty directory."""
    monkeypatch.setattr(config, "CONFIG_FILE_PATH", tmp_path)
    monkeypatch.setattr(config, "DESKTOP_FILE_PATH", tmp_path / "desktop.json")
    return tmp_path


class TestLoadDesktopOptions:
    def test_reads_the_desktop_file(self, tmp_path: Path) -> None:
        """Should read the address and the API key that Synnax Desktop saved."""
        path = tmp_path / "desktop.json"
        write_desktop_file(path, port=4321)
        assert load_desktop_options(path) == Options(
            host="127.0.0.1", port=4321, secure=False, api_key="syk_k_s"
        )

    def test_none_without_a_file(self, tmp_path: Path) -> None:
        """Should return None when Synnax Desktop saved nothing."""
        assert load_desktop_options(tmp_path / "desktop.json") is None


class TestTryLoadOptions:
    def test_given_options(self, home: Path) -> None:
        """Should use the given options and not read the Desktop file."""
        write_desktop_file(home / "desktop.json", port=4321)
        opts, embedded = try_load_options_if_none_provided(host="h", port=1)
        assert (opts.host, opts.port, embedded) == ("h", 1, False)

    def test_desktop_without_a_saved_login(self, home: Path) -> None:
        """Should use the Desktop file when `sy login` saved nothing."""
        write_desktop_file(home / "desktop.json", port=4321)
        opts, embedded = try_load_options_if_none_provided()
        assert (opts.port, opts.api_key, embedded) == (4321, "syk_k_s", True)

    def test_nothing_saved(self, home: Path) -> None:
        """Should explain how to connect when nothing is saved."""
        with pytest.raises(ValidationError, match=NO_OPTIONS_MSG):
            try_load_options_if_none_provided()


class TestDesktopConnection:
    def test_not_running(self, home: Path) -> None:
        """Should say that Synnax Desktop is not running when its Core is gone."""
        write_desktop_file(home / "desktop.json", port=closed_port())
        with pytest.raises(Unreachable) as exc:
            sy.Synnax(max_retries=0)
        assert str(exc.value) == DESKTOP_NOT_RUNNING_MSG
