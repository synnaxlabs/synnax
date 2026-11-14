#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from pathlib import Path
from types import SimpleNamespace

import pytest

import agent_boot
from agent_boot import COMMIT_PARAMETER, FILES, INSTALLED, SOURCE, update

COMMIT = "a" * 40
OLD = "b" * 40


class Store:
    """Stands in for Parameter Store: serves the deployed commit."""

    def __init__(self) -> None:
        self.requests: list[str] = []

    def get_parameter(self, *, Name: str) -> dict:
        self.requests.append(Name)
        return {"Parameter": {"Value": COMMIT}}


class GitHub:
    """Stands in for GitHub: serves a URL as the content of its file, and fails for the
    file names in missing."""

    def __init__(self) -> None:
        self.urls: list[str] = []
        self.missing: set[str] = set()

    def __call__(self, url: str) -> bytes:
        self.urls.append(url)
        if url.rsplit("/", 1)[1] in self.missing:
            raise OSError("HTTP Error 404: Not Found")
        return url.encode()


class TestUpdate:
    """Tests for bringing the agent files to the deployed commit."""

    def test_should_install_the_agent_of_the_deployed_commit(
        self, tmp_path: Path
    ) -> None:
        store, github = Store(), GitHub()
        assert update(tmp_path, store=store, fetch=github) == COMMIT
        assert store.requests == [COMMIT_PARAMETER]
        for name in FILES:
            assert (tmp_path / name).read_text() == SOURCE.format(COMMIT, name)
        assert (tmp_path / INSTALLED).read_text() == COMMIT

    def test_should_replace_the_agent_of_another_commit(self, tmp_path: Path) -> None:
        (tmp_path / INSTALLED).write_text(OLD)
        (tmp_path / "agent.py").write_text("old")
        assert update(tmp_path, store=Store(), fetch=GitHub()) == COMMIT
        assert (tmp_path / "agent.py").read_text() != "old"

    def test_should_leave_an_agent_that_is_up_to_date(self, tmp_path: Path) -> None:
        (tmp_path / INSTALLED).write_text(f"{COMMIT}\n")
        github = GitHub()
        assert update(tmp_path, store=Store(), fetch=github) is None
        assert github.urls == []

    def test_should_keep_the_old_agent_when_a_download_fails(
        self, tmp_path: Path
    ) -> None:
        (tmp_path / INSTALLED).write_text(OLD)
        (tmp_path / "agent.py").write_text("old")
        github = GitHub()
        github.missing = {FILES[-1]}
        with pytest.raises(OSError, match="404"):
            update(tmp_path, store=Store(), fetch=github)
        assert (tmp_path / "agent.py").read_text() == "old"
        assert (tmp_path / INSTALLED).read_text() == OLD


class Response:
    def __init__(self, body: bytes) -> None:
        self.body = body

    def __enter__(self) -> "Response":
        return self

    def __exit__(self, *args: object) -> None:
        return None

    def read(self) -> bytes:
        return self.body


class Machine:
    """Stands in for the runner: records each command and returns the given code."""

    def __init__(self) -> None:
        self.commands: list[list[str]] = []
        self.code = 0

    def run(self, command: list[str], *, check: bool) -> SimpleNamespace:
        self.commands.append(command)
        return SimpleNamespace(returncode=self.code)


@pytest.fixture
def machine(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Machine:
    """A Linux runner whose boot script is in tmp_path, started with two arguments."""
    m = Machine()
    argv = ["agent_boot.py", "--host", "core.example"]
    monkeypatch.setattr(agent_boot, "__file__", str(tmp_path / "agent_boot.py"))
    monkeypatch.setattr(agent_boot, "sys", SimpleNamespace(platform="linux", argv=argv))
    monkeypatch.setattr(agent_boot.subprocess, "run", m.run)
    monkeypatch.setattr(agent_boot.boto3, "client", lambda *a, **k: Store())
    monkeypatch.setattr(
        agent_boot, "urlopen", lambda url, timeout: Response(url.encode())
    )
    return m


class TestMain:
    """Tests for the start of the agent on a runner."""

    def test_should_update_then_run_the_agent_with_its_own_arguments(
        self, machine: Machine, tmp_path: Path, capsys: pytest.CaptureFixture[str]
    ) -> None:
        machine.code = 3
        assert agent_boot.main() == 3
        agent = tmp_path / "agent.py"
        assert agent.read_text() == SOURCE.format(COMMIT, "agent.py")
        assert machine.commands == [
            [str(tmp_path / "uv"), "run", str(agent), "--host", "core.example"]
        ]
        assert capsys.readouterr().out == f"installed the agent at {COMMIT}\n"

    def test_should_run_the_installed_agent_when_the_update_fails(
        self,
        machine: Machine,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        def refuse(*args: object, **kwargs: object) -> None:
            raise PermissionError("AccessDenied")

        monkeypatch.setattr(agent_boot.boto3, "client", refuse)
        assert agent_boot.main() == 0
        assert len(machine.commands) == 1
        assert capsys.readouterr().out == (
            "update failed: PermissionError: AccessDenied\n"
        )

    def test_should_run_the_windows_uv_on_windows(
        self, machine: Machine, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(
            agent_boot, "sys", SimpleNamespace(platform="win32", argv=[])
        )
        agent_boot.main()
        assert machine.commands[0][0] == str(tmp_path / "uv.exe")
