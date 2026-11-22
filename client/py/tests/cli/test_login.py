#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import builtins
import getpass
import io
import socket
import sys
from pathlib import Path

import pytest

from synnax import config
from synnax.cli import login
from synnax.config import Clusters, File
from tests.memory_keyring import MemoryKeyring


@pytest.mark.cli
class TestLogin:
    def test_masks_the_password_in_a_frontend_without_a_tty(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        """Should ask for the password through getpass when a frontend such as
        Jupyter replaces input and getpass, and stdin is not a terminal."""
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            port = s.getsockname()[1]
        plain: list[str] = []
        masked: list[str] = []

        def frontend_input(prompt: str = "") -> str:
            plain.append(prompt)
            return "secret"

        def frontend_getpass(prompt: str = "Password: ") -> str:
            masked.append(prompt)
            return "secret"

        clusters = Clusters(File(config_dir=tmp_path), MemoryKeyring())
        monkeypatch.setattr(config, "default_clusters", lambda: clusters)
        monkeypatch.setattr(
            sys, "stdin", io.StringIO(f"127.0.0.1\n{port}\nsynnax\nn\n")
        )
        monkeypatch.setattr(builtins, "input", frontend_input)
        monkeypatch.setattr(getpass, "getpass", frontend_getpass)
        assert not login.login()
        assert masked == ["Password: "]
        assert plain == []
