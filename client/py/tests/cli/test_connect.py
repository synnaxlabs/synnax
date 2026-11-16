#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
from pathlib import Path

import pytest

from synnax.cli import connect, console
from synnax.cli.flow import Context
from synnax.config import Clusters, File
from tests.memory_keyring import MemoryKeyring


@pytest.fixture
def clusters(tmp_path: Path) -> Clusters:
    return Clusters(File(config_dir=tmp_path), MemoryKeyring())


@pytest.fixture
def closed_port() -> int:
    with socket.socket() as s:
        s.bind(("127.0.0.1", 0))
        port: int = s.getsockname()[1]
    return port


@pytest.mark.cli
class TestPromptedConnection:
    def test_connects_to_prompted_address(
        self, clusters: Clusters, tmp_path: Path, closed_port: int
    ) -> None:
        """Should connect to the host and port from the prompt responses."""
        mock = console.Mock(
            responses=["127.0.0.1", closed_port, "admin", "secret", False]
        )
        desktop = tmp_path / "desktop.json"
        assert connect.from_clusters(Context(console=mock), clusters, desktop) is None
        messages = [e.message or "" for e in mock.output.entries]
        assert not any("CA certificate" in m for m in messages)
        assert messages[-1] == f"Cannot reach Synnax server at 127.0.0.1:{closed_port}"

    @pytest.mark.parametrize("contents", [None, "not a certificate"])
    def test_bad_ca_cert(
        self,
        clusters: Clusters,
        tmp_path: Path,
        closed_port: int,
        monkeypatch: pytest.MonkeyPatch,
        contents: str | None,
    ) -> None:
        """Should report a missing or invalid CA certificate by its absolute path
        instead of raising."""
        ca_cert = tmp_path / "ca.crt"
        if contents is not None:
            ca_cert.write_text(contents)
        monkeypatch.chdir(tmp_path)
        mock = console.Mock(
            responses=["127.0.0.1", closed_port, "admin", "secret", True, "ca.crt"]
        )
        desktop = tmp_path / "desktop.json"
        assert connect.from_clusters(Context(console=mock), clusters, desktop) is None
        assert str(ca_cert.resolve()) in (mock.output.entries[-1].message or "")
