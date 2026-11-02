#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.


import json
import os
import socket
import subprocess
import sys
from pathlib import Path

import pytest

from synnax.config import (
    DESKTOP_NOT_RUNNING_MSG,
    NO_OPTIONS_MSG,
    Cluster,
    Clusters,
    File,
    load_desktop_options,
    try_load_options_if_none_provided,
)
from synnax.exceptions import ValidationError
from synnax.options import Options
from tests.memory_keyring import MemoryKeyring


def write_desktop_file(path: Path, *, port: int, api_key: str = "syk_k_s") -> None:
    """Writes the file the way Synnax Desktop does."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(
            {"host": "127.0.0.1", "port": port, "secure": False, "api_key": api_key}
        )
    )


@pytest.fixture
def clusters(tmp_path: Path) -> Clusters:
    return Clusters(File(config_dir=tmp_path), MemoryKeyring())


@pytest.fixture
def desktop_file(tmp_path: Path) -> Path:
    return tmp_path / "desktop.json"


class TestLoadDesktopOptions:
    def test_reads_the_desktop_file(self, desktop_file: Path) -> None:
        """Should read the address and the API key that Synnax Desktop saved."""
        write_desktop_file(desktop_file, port=4321)
        assert load_desktop_options(desktop_file) == Options(
            host="127.0.0.1", port=4321, secure=False, api_key="syk_k_s"
        )

    def test_none_without_a_file(self, desktop_file: Path) -> None:
        """Should return None when Synnax Desktop saved nothing."""
        assert load_desktop_options(desktop_file) is None


class TestTryLoadOptions:
    def test_given_options(self, clusters: Clusters, desktop_file: Path) -> None:
        """Should use the given options, which Synnax Desktop did not save."""
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            host="h", port=1, clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.host, opts.port, embedded) == ("h", 1, False)

    def test_given_desktop_options(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should know the options that Synnax Desktop saved when a caller gives them,
        as the CLI does.
        """
        write_desktop_file(desktop_file, port=4321)
        _, embedded = try_load_options_if_none_provided(
            host="127.0.0.1",
            port=4321,
            api_key="syk_k_s",
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert embedded

    def test_given_options_with_a_broken_desktop_file(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should connect with the given options when the Desktop file is broken."""
        desktop_file.write_text("not json")
        opts, embedded = try_load_options_if_none_provided(
            host="h",
            port=1,
            api_key="syk_k_s",
            clusters=clusters,
            desktop_file=desktop_file,
        )
        assert (opts.host, embedded) == ("h", False)

    def test_saved_login_before_desktop(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the login that `sy login` saved before the Desktop file."""
        clusters.set(Cluster(options=Options(host="h", port=1, api_key="syk_saved")))
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.api_key, embedded) == ("syk_saved", False)

    def test_desktop_without_a_saved_login(
        self, clusters: Clusters, desktop_file: Path
    ) -> None:
        """Should use the Desktop file when `sy login` saved nothing."""
        write_desktop_file(desktop_file, port=4321)
        opts, embedded = try_load_options_if_none_provided(
            clusters=clusters, desktop_file=desktop_file
        )
        assert (opts.port, opts.api_key, embedded) == (4321, "syk_k_s", True)

    def test_nothing_saved(self, clusters: Clusters, desktop_file: Path) -> None:
        """Should explain how to connect when nothing is saved."""
        with pytest.raises(ValidationError, match=NO_OPTIONS_MSG):
            try_load_options_if_none_provided(
                clusters=clusters, desktop_file=desktop_file
            )


class TestDesktopConnection:
    def test_not_running(self, tmp_path: Path) -> None:
        """Should say that Synnax Desktop is not running when its Core is gone."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = int(s.getsockname()[1])
        write_desktop_file(tmp_path / ".synnax" / "desktop.json", port=port)
        # A fresh interpreter reads the home directory of the test.
        env = {**os.environ, "HOME": str(tmp_path), "USERPROFILE": str(tmp_path)}
        result = subprocess.run(
            [sys.executable, "-c", "import synnax as sy; sy.Synnax(max_retries=0)"],
            env=env,
            capture_output=True,
            text=True,
            timeout=30,
        )
        assert result.returncode != 0
        assert DESKTOP_NOT_RUNNING_MSG in result.stderr
