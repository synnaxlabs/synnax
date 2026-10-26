#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import subprocess
import sys
import time
from pathlib import Path

import pytest

from agent.agent import (
    Options,
    agent_command,
    branch_name,
    default_title,
    git_auth,
    instructions,
    main,
    parse,
    render_body,
    run_agent,
    skill_prompt,
    stored_names,
)

TEMPLATE = """## Linear issue

<!-- Edit the link below with the proper issue number and link -->

[SY-####](<>)

## Description

<!-- What changed and why. -->

## Review tier

<!-- Add one label. -->
"""


class TestParse:
    def test_agent_defaults_from_environment(self) -> None:
        opts = parse(["--prompt", "fix it"], {"AGENT_DEFAULT": "codex"})
        assert opts.agent == "codex"
        assert opts.output == "pr"

    def test_flag_overrides_environment(self) -> None:
        opts = parse(["--prompt", "x", "--agent", "claude"], {"AGENT_DEFAULT": "codex"})
        assert opts.agent == "claude"

    def test_rejects_missing_agent(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x"], {})

    def test_rejects_comment_without_pr(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x", "--agent", "claude", "--output", "comment"], {})

    def test_rejects_turn_limit_for_codex(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x", "--agent", "codex", "--max-turns", "5"], {})

    def test_splits_labels(self) -> None:
        argv = ["--prompt", "x", "--agent", "claude", "--labels", "a, b,,c"]
        assert parse(argv, {}).labels == ["a", "b", "c"]

    def test_reads_prompt_file(self, tmp_path: Path) -> None:
        f = tmp_path / "prompt.md"
        f.write_text("  do the thing\n")
        opts = parse(["--prompt-file", str(f), "--agent", "claude"], {})
        assert opts.prompt == "do the thing"


class TestNames:
    def test_branch_name(self) -> None:
        name = branch_name("claude", "SY-12: Fix the flaky Timer spec!", "a1b2c3")
        assert name == "agent/claude/sy-12-fix-the-flaky-timer-a1b2c3"

    def test_default_title_strips_slash_and_truncates(self) -> None:
        assert default_title("/flake-hunt core\nmore") == "[agent] flake-hunt core"
        assert len(default_title("x" * 200)) == 72


class TestSkillPrompt:
    def test_spells_out_known_skill(self, tmp_path: Path) -> None:
        skill = tmp_path / ".claude/skills/flake-hunt/SKILL.md"
        skill.parent.mkdir(parents=True)
        skill.write_text("x")
        got = skill_prompt("/flake-hunt core", tmp_path)
        assert got == (
            "Follow the instructions in `.claude/skills/flake-hunt/SKILL.md`."
            " Arguments: core"
        )

    def test_keeps_unknown_skill_and_plain_prompts(self, tmp_path: Path) -> None:
        assert skill_prompt("/nope", tmp_path) == "/nope"
        assert skill_prompt("fix it", tmp_path) == "fix it"


class TestAgentCommand:
    def test_claude(self, tmp_path: Path) -> None:
        opts = Options("p", "claude", "pr", model="opus", max_turns=40)
        cmd = agent_command(opts, "p", tmp_path)
        assert cmd[:2] == ["claude", "-p"]
        assert cmd[-1] == "p"
        assert "--dangerously-skip-permissions" in cmd
        assert cmd[cmd.index("--max-turns") + 1] == "40"
        assert cmd[cmd.index("--model") + 1] == "opus"

    def test_codex(self, tmp_path: Path) -> None:
        cmd = agent_command(Options("p", "codex", "report"), "p", tmp_path)
        assert cmd[:2] == ["codex", "exec"]
        assert cmd[-1] == "p"
        assert "--model" not in cmd
        assert cmd[cmd.index("--output-last-message") + 1].endswith("last-message.md")


class TestInstructions:
    def test_comment_on_pr_commits_to_it(self, tmp_path: Path) -> None:
        text = instructions(Options("p", "claude", "comment", pr=12), tmp_path)
        assert "it is pushed to the PR" in text
        assert "title.txt" not in text

    def test_new_pr_asks_for_title_and_description(self, tmp_path: Path) -> None:
        text = instructions(Options("p", "claude", "pr"), tmp_path)
        assert "title.txt" in text
        assert "description.md" in text


class TestRenderBody:
    def test_fills_template(self) -> None:
        body = render_body(TEMPLATE, "SY-12", "Fixed it.", ["bug", "review/bot"])
        assert "<!--" not in body
        assert "[SY-12](https://linear.app/synnax/issue/SY-12)" in body
        assert "## Description\n\nFixed it.\n" in body
        assert body.endswith("## Review tier\n\n`review/bot`\n")

    def test_no_issue(self) -> None:
        assert "## Linear issue\n\nNone\n" in render_body(TEMPLATE, "", "d", [])


FAKE_AGENT = """#!/usr/bin/env python3
import os, pathlib
out = pathlib.Path(os.environ["AGENT_OUT_DIR"])
pathlib.Path("fixed.txt").write_text("fixed")
(out / "title.txt").write_text("SY-1: Fix the thing")
(out / "issue.txt").write_text("SY-1")
(out / "labels.txt").write_text("review/bot")
(out / "images" / "shot.png").write_bytes(b"png")
print("Done.")
"""


@pytest.fixture
def repo(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> Path:
    """A clean repository with the PR template, as the working directory."""
    root = tmp_path / "repo"
    (root / ".github").mkdir(parents=True)
    (root / ".github/pull_request_template.md").write_text(TEMPLATE)
    git(root, "init", "--quiet", "--initial-branch", "main")
    git(root, "add", "--all")
    git(root, "-c", "user.name=t", "-c", "user.email=t@t", "commit", "-qm", "i")
    monkeypatch.chdir(root)
    monkeypatch.setenv("AGENT_GIT_NAME", "Synnax Claude Code[bot]")
    monkeypatch.setenv("AGENT_GIT_EMAIL", "1+bot@users.noreply.github.com")
    monkeypatch.setenv("GH_TOKEN", "token")
    monkeypatch.delenv("GITHUB_OUTPUT", raising=False)
    return root


def git(root: Path, *args: str) -> str:
    cmd = ["git", "-C", str(root), *args]
    return subprocess.check_output(cmd, text=True).strip()


def fake_agent(tmp_path: Path) -> list[str]:
    fake = tmp_path / "fake-agent"
    fake.write_text(FAKE_AGENT)
    fake.chmod(0o755)
    return ["--prompt", "Fix the thing", "--agent", "claude", "--bin", str(fake)]


class TestMain:
    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_dry_run_opens_nothing(
        self, repo: Path, tmp_path: Path, capsys: pytest.CaptureFixture[str]
    ) -> None:
        assert main([*fake_agent(tmp_path), "--dry-run"]) == 0
        branch = git(repo, "branch", "--show-current")
        assert branch.startswith("agent/claude/fix-the-thing-")
        author = git(repo, "log", "-1", "--format=%an|%s")
        assert author == "Synnax Claude Code[bot]|SY-1: Fix the thing"
        err = capsys.readouterr().err
        assert "dry run: git push origin" in err
        assert "dry run: gh pr create" in err
        assert "[SY-1](https://linear.app/synnax/issue/SY-1)" in err
        assert "![1-shot.png](https://raw.githubusercontent.com/" in err

    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_report_is_a_step_output(
        self, repo: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        output = tmp_path / "github_output"
        monkeypatch.setenv("GITHUB_OUTPUT", str(output))
        assert main([*fake_agent(tmp_path), "--output", "report"]) == 0
        lines = output.read_text().splitlines()
        assert lines[0].startswith("report<<EOF_")
        assert lines[1:] == ["Done.", lines[0].removeprefix("report<<")]

    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_no_commit_opens_no_pr(
        self,
        repo: Path,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        idle = tmp_path / "idle-agent"
        idle.write_text("#!/bin/sh\necho Nothing to fix.\n")
        idle.chmod(0o755)
        output = tmp_path / "github_output"
        monkeypatch.setenv("GITHUB_OUTPUT", str(output))
        argv = ["--prompt", "Fix it", "--agent", "claude", "--bin", str(idle)]
        assert main(argv) == 0
        assert "The agent made no changes" in capsys.readouterr().out
        assert not output.exists()
        assert git(repo, "rev-list", "--count", "HEAD") == "1"

    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_report_agent_may_push_its_own_work(
        self,
        repo: Path,
        tmp_path: Path,
        capsys: pytest.CaptureFixture[str],
    ) -> None:
        agent = tmp_path / "env-agent"
        agent.write_text('#!/bin/sh\necho "auth=$GIT_CONFIG_KEY_0"\n')
        agent.chmod(0o755)
        argv = ["--prompt", "Hunt", "--agent", "claude", "--bin", str(agent)]
        assert main([*argv, "--output", "report"]) == 0
        assert "auth=http.https://github.com/.extraheader" in capsys.readouterr().out
        text = instructions(Options("p", "claude", "report"), tmp_path)
        assert "Never push" not in text
        assert main([*argv, "--output", "report", "--dry-run"]) == 0
        assert "auth=\n" in capsys.readouterr().out
        dry = instructions(Options("p", "claude", "report", dry_run=True), tmp_path)
        assert "never push" in dry

    def test_rejects_dirty_checkout(self, repo: Path, tmp_path: Path) -> None:
        (repo / "stray.txt").write_text("not the agent's")
        with pytest.raises(SystemExit, match="uncommitted changes"):
            main(fake_agent(tmp_path))

    def test_rejects_missing_bot_identity(
        self, repo: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("GIT_AUTHOR_NAME", "A Person")
        monkeypatch.delenv("AGENT_GIT_EMAIL")
        with pytest.raises(SystemExit, match="bot identity"):
            main(fake_agent(tmp_path))


def test_stored_names_are_unique_and_url_safe() -> None:
    paths = [Path("before/shot.png"), Path("after/shot.png"), Path("a#1 [x].png")]
    assert stored_names(paths) == ["1-shot.png", "2-shot.png", "3-a_1__x_.png"]


@pytest.mark.skipif(sys.platform == "win32", reason="uses a POSIX shell")
def test_timeout_kills_the_agent_and_its_children(tmp_path: Path) -> None:
    pid_file = tmp_path / "pid"
    script = f"sleep 60 & echo $! > {pid_file}; wait"
    with pytest.raises(subprocess.TimeoutExpired):
        run_agent(["sh", "-c", script], {"PATH": "/usr/bin:/bin"}, 1)
    child = pid_file.read_text().strip()
    # The orphaned child may linger as a zombie until init reaps it.
    for _ in range(50):
        ps = subprocess.run(["ps", "-o", "stat=", "-p", child], capture_output=True)
        if not ps.stdout.strip() or ps.stdout.startswith(b"Z"):
            break
        time.sleep(0.1)
    else:
        pytest.fail(f"process {child} still runs")


def test_git_auth_sends_the_app_token() -> None:
    env = git_auth({"GH_TOKEN": "abc"})
    assert env["GIT_CONFIG_KEY_0"] == "http.https://github.com/.extraheader"
    encoded = env["GIT_CONFIG_VALUE_0"].removeprefix("AUTHORIZATION: basic ")
    assert base64.b64decode(encoded) == b"x-access-token:abc"
