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

import agent_log
from agent_log import Log
from conftest import Client, core

CHANNEL = "ubuntu_build_i_0f44_log"


@pytest.fixture
def page(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    monkeypatch.setattr(agent_log, "LOG_DIRS", (tmp_path,))
    return tmp_path / "Runner_1.log"


def append(page: Path, text: str) -> None:
    with page.open("a") as f:
        f.write(text)


class TestLog:
    """Tests for tailing the runner page logs into the log channel."""

    def test_should_skip_the_lines_written_before_it_started(
        self, client: Client, page: Path
    ) -> None:
        page.write_text("old\n")
        Log(core(client), CHANNEL).write()
        assert client.written(CHANNEL) == []

    def test_should_send_the_lines_written_after_it_started(
        self, client: Client, page: Path
    ) -> None:
        page.write_text("old\n")
        log = Log(core(client), CHANNEL)
        append(page, "one\ntwo\n")
        log.write()
        log.write()
        assert client.written(CHANNEL) == [["one", "two"]]

    def test_should_hold_a_partial_line_until_its_newline(
        self, client: Client, page: Path
    ) -> None:
        log = Log(core(client), CHANNEL)
        append(page, "par")
        log.write()
        append(page, "tial\n")
        log.write()
        assert client.written(CHANNEL) == [["partial"]]

    def test_should_read_a_truncated_file_from_its_start(
        self, client: Client, page: Path
    ) -> None:
        page.write_text("a long first line\n")
        log = Log(core(client), CHANNEL)
        page.write_text("new\n")
        log.write()
        assert client.written(CHANNEL) == [["new"]]

    def test_should_read_a_replaced_file_from_its_start(
        self, client: Client, page: Path
    ) -> None:
        page.write_text("old line\n")
        log = Log(core(client), CHANNEL)
        page.unlink()
        log.write()
        page.write_text("a longer new line\n")
        log.write()
        assert client.written(CHANNEL) == [["a longer new line"]]

    def test_should_read_at_most_the_budget_in_one_cycle(
        self, client: Client, page: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setattr(agent_log, "LOG_BUDGET", 8)
        log = Log(core(client), CHANNEL)
        append(page, "one\ntwo\nthree\n")
        log.write()
        log.write()
        assert client.written(CHANNEL) == [["one", "two"], ["three"]]

    def test_should_skip_a_file_it_cannot_open(
        self, client: Client, page: Path
    ) -> None:
        log = Log(core(client), CHANNEL)
        page.with_name("Worker_1.log").mkdir()
        append(page, "one\n")
        log.write()
        assert client.written(CHANNEL) == [["one"]]

    def test_should_send_the_lines_of_the_agent_first(
        self, client: Client, page: Path, capsys: pytest.CaptureFixture[str]
    ) -> None:
        log = Log(core(client), CHANNEL)
        append(page, "runner\n")
        log.log("agent connected")
        log.write()
        assert client.written(CHANNEL) == [["[agent] agent connected", "runner"]]
        assert capsys.readouterr().out == "agent connected\n"

    def test_should_keep_the_lines_of_the_agent_when_a_write_fails(
        self, client: Client, page: Path
    ) -> None:
        log = Log(core(client), CHANNEL)
        log.log("agent connected")
        client.failing = True
        with pytest.raises(ConnectionError):
            log.write()
        client.failing = False
        log.write()
        assert client.written(CHANNEL) == [["[agent] agent connected"]]
