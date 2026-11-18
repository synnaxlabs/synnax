#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import socket
import threading
from collections.abc import Iterator
from pathlib import Path

import pytest

from synnax.cli import connect
from synnax.config import Clusters, File
from tests.cli.scripted import Scripted
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


@pytest.fixture
def silent_port() -> Iterator[int]:
    """A port that accepts a connection and never responds."""
    listener = socket.create_server(("127.0.0.1", 0))

    def hold() -> None:
        conn, _ = listener.accept()
        with conn:
            while conn.recv(1 << 16):
                pass

    thread = threading.Thread(target=hold, daemon=True)
    thread.start()
    with listener:
        yield int(listener.getsockname()[1])
        thread.join(10)


@pytest.mark.cli
class TestPromptedConnection:
    def test_connects_to_prompted_address(
        self, clusters: Clusters, closed_port: int
    ) -> None:
        """Should connect to the host and port from the answers."""
        user = Scripted("127.0.0.1", closed_port, "admin", "secret", "n")
        assert not connect.login(user.prompt, clusters)
        assert not any("CA certificate" in q for q in user.questions)
        assert user.err.getvalue().splitlines()[-1] == (
            f"Cannot reach Synnax server at 127.0.0.1:{closed_port}"
        )
        assert clusters.get() is None

    def test_no_response(self, clusters: Clusters, silent_port: int) -> None:
        """Should say that the server sent no response when it does not respond in
        time."""
        user = Scripted("127.0.0.1", silent_port, "admin", "secret", "n")
        assert not connect.login(user.prompt, clusters)
        assert user.err.getvalue().splitlines()[-1] == (
            f"No response from Synnax server at 127.0.0.1:{silent_port}"
        )
        assert clusters.get() is None

    def test_defaults(self, clusters: Clusters) -> None:
        """Should use the default host, port, and username for empty answers."""
        user = Scripted("", "", "", "secret", "")
        assert not connect.login(user.prompt, clusters)
        assert user.questions == [
            "Host [localhost]: ",
            "Port [9090]: ",
            "Username [synnax]: ",
            "Password: ",
            "Secure connection? [y/N]: ",
        ]

    def test_asks_again_for_a_port_that_is_not_an_integer(
        self, clusters: Clusters, closed_port: int
    ) -> None:
        """Should ask for the port again until the answer is an integer."""
        user = Scripted("127.0.0.1", "abc", closed_port, "admin", "secret", "n")
        assert not connect.login(user.prompt, clusters)
        assert user.questions.count("Port [9090]: ") == 2
        assert "'abc' is not an integer." in user.err.getvalue()

    def test_asks_again_for_an_unknown_yes_or_no(
        self, clusters: Clusters, closed_port: int
    ) -> None:
        """Should ask the secure question again until the answer is yes or no."""
        user = Scripted("127.0.0.1", closed_port, "admin", "secret", "maybe", "n")
        assert not connect.login(user.prompt, clusters)
        assert user.questions.count("Secure connection? [y/N]: ") == 2

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
        user = Scripted("127.0.0.1", closed_port, "admin", "secret", "y", "ca.crt")
        assert not connect.login(user.prompt, clusters)
        assert user.questions[-1] == "CA certificate path (empty for system roots): "
        assert str(ca_cert.resolve()) in user.err.getvalue().splitlines()[-1]
