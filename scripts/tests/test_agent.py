#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import base64
import shutil
import subprocess
import sys
import tempfile
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
    """Tests for parsing the command line."""

    def test_should_take_the_agent_from_the_environment(self) -> None:
        opts = parse(["--prompt", "fix it"], env={"AGENT_DEFAULT": "codex"})
        assert opts.agent == "codex"
        assert opts.output == "pr"

    def test_should_prefer_the_flag_to_the_environment(self) -> None:
        opts = parse(
            ["--prompt", "x", "--agent", "claude"], env={"AGENT_DEFAULT": "codex"}
        )
        assert opts.agent == "claude"

    def test_should_reject_a_missing_agent(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x"], env={})

    def test_should_reject_a_comment_without_a_pr(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x", "--agent", "claude", "--output", "comment"], env={})

    def test_should_reject_a_recipient_without_a_report(self) -> None:
        with pytest.raises(SystemExit):
            parse(
                ["--prompt", "x", "--agent", "claude", "--recipient", "k.asc"], env={}
            )

    def test_should_reject_a_turn_limit_for_codex(self) -> None:
        with pytest.raises(SystemExit):
            parse(["--prompt", "x", "--agent", "codex", "--max-turns", "5"], env={})

    def test_should_split_labels(self) -> None:
        argv = ["--prompt", "x", "--agent", "claude", "--labels", "a, b,,c"]
        assert parse(argv, env={}).labels == ["a", "b", "c"]

    def test_should_read_a_prompt_file(self, tmp_path: Path) -> None:
        f = tmp_path / "prompt.md"
        f.write_text("  do the thing\n")
        opts = parse(["--prompt-file", str(f), "--agent", "claude"], env={})
        assert opts.prompt == "do the thing"


class TestNames:
    """Tests for branch names and default titles."""

    def test_should_name_the_branch_after_agent_and_title(self) -> None:
        name = branch_name(
            agent="claude", title="SY-12: Fix the flaky Timer spec!", suffix="a1b2c3"
        )
        assert name == "agent/claude/sy-12-fix-the-flaky-timer-a1b2c3"

    def test_should_strip_the_slash_and_truncate_the_title(self) -> None:
        assert default_title("/flake-hunt core\nmore") == "[agent] flake-hunt core"
        assert len(default_title("x" * 200)) == 72


class TestSkillPrompt:
    """Tests for spelling out slash skills for Codex."""

    def test_should_spell_out_a_known_skill(self, tmp_path: Path) -> None:
        skill = tmp_path / ".claude/skills/flake-hunt/SKILL.md"
        skill.parent.mkdir(parents=True)
        skill.write_text("x")
        got = skill_prompt("/flake-hunt core", root=tmp_path)
        assert got == (
            "Follow the instructions in `.claude/skills/flake-hunt/SKILL.md`."
            " Arguments: core"
        )

    def test_should_keep_unknown_skills_and_plain_prompts(self, tmp_path: Path) -> None:
        assert skill_prompt("/nope", root=tmp_path) == "/nope"
        assert skill_prompt("fix it", root=tmp_path) == "fix it"


class TestAgentCommand:
    """Tests for the command line each agent CLI gets."""

    def test_should_build_a_claude_command(self, tmp_path: Path) -> None:
        opts = Options(
            prompt="p", agent="claude", output="pr", model="opus", max_turns=40
        )
        cmd = agent_command(opts, prompt="p", out=tmp_path)
        assert cmd[:2] == ["claude", "-p"]
        assert cmd[-1] == "p"
        assert "--dangerously-skip-permissions" in cmd
        assert cmd[cmd.index("--max-turns") + 1] == "40"
        assert cmd[cmd.index("--model") + 1] == "opus"

    def test_should_build_a_codex_command(self, tmp_path: Path) -> None:
        cmd = agent_command(
            Options(prompt="p", agent="codex", output="report"),
            prompt="p",
            out=tmp_path,
        )
        assert cmd[:2] == ["codex", "exec"]
        assert cmd[-1] == "p"
        assert "--model" not in cmd
        assert cmd[cmd.index("--output-last-message") + 1].endswith("last-message.md")


class TestInstructions:
    """Tests for the delivery instructions given to the agent."""

    def test_should_commit_to_the_pr_when_commenting(self, tmp_path: Path) -> None:
        text = instructions(
            Options(prompt="p", agent="claude", output="comment", pr=12), out=tmp_path
        )
        assert "it is pushed to the PR" in text
        assert "title.txt" not in text

    def test_should_ask_for_a_title_and_description(self, tmp_path: Path) -> None:
        text = instructions(
            Options(prompt="p", agent="claude", output="pr"), out=tmp_path
        )
        assert "title.txt" in text
        assert "description.md" in text


class TestRenderBody:
    """Tests for filling the pull request template."""

    def test_should_fill_the_template(self) -> None:
        body = render_body(
            TEMPLATE,
            issue="SY-12",
            description="Fixed it.",
            labels=["bug", "review/bot"],
        )
        assert "<!--" not in body
        assert "[SY-12](https://linear.app/synnax/issue/SY-12)" in body
        assert "## Description\n\nFixed it.\n" in body
        assert body.endswith("## Review tier\n\n`review/bot`\n")

    def test_should_write_none_without_an_issue(self) -> None:
        assert "## Linear issue\n\nNone\n" in render_body(
            TEMPLATE, issue="", description="d", labels=[]
        )


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
    """Tests for whole runs against a fake agent."""

    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_should_open_nothing_in_a_dry_run(
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
    def test_should_write_the_report_as_a_step_output(
        self, repo: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        output = tmp_path / "github_output"
        monkeypatch.setenv("GITHUB_OUTPUT", str(output))
        assert main([*fake_agent(tmp_path), "--output", "report"]) == 0
        lines = output.read_text().splitlines()
        assert lines[0].startswith("report<<EOF_")
        assert lines[1:] == ["Done.", lines[0].removeprefix("report<<")]

    @pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
    def test_should_open_no_pr_without_a_commit(
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
    def test_should_let_a_report_agent_push(
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
        text = instructions(
            Options(prompt="p", agent="claude", output="report"), out=tmp_path
        )
        assert "Never push" not in text
        assert main([*argv, "--output", "report", "--dry-run"]) == 0
        assert "auth=\n" in capsys.readouterr().out
        dry = instructions(
            Options(prompt="p", agent="claude", output="report", dry_run=True),
            out=tmp_path,
        )
        assert "never push" in dry

    def test_should_reject_a_dirty_checkout(self, repo: Path, tmp_path: Path) -> None:
        (repo / "stray.txt").write_text("not the agent's")
        with pytest.raises(SystemExit, match="uncommitted changes"):
            main(fake_agent(tmp_path))

    def test_should_reject_a_missing_bot_identity(
        self, repo: Path, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        monkeypatch.setenv("GIT_AUTHOR_NAME", "A Person")
        monkeypatch.delenv("AGENT_GIT_EMAIL")
        with pytest.raises(SystemExit, match="bot identity"):
            main(fake_agent(tmp_path))


PRIVATE_AGENT = """#!/bin/sh
echo "secret on stderr" >&2
echo "secret report token=$GH_TOKEN summary=$GITHUB_STEP_SUMMARY \
login=$(cat "$CLAUDE_CONFIG_DIR/.credentials.json")"
echo "$CLAUDE_CONFIG_DIR" > "$STATE_FILE"
"""


@pytest.mark.skipif(shutil.which("gpg") is None, reason="needs gpg")
@pytest.mark.skipif(sys.platform == "win32", reason="fake agent is a script")
class TestPrivateReport:
    """Tests for a report encrypted to a recipient."""

    def test_should_write_only_the_encrypted_report(
        self,
        repo: Path,
        tmp_path: Path,
        monkeypatch: pytest.MonkeyPatch,
        capfd: pytest.CaptureFixture[str],
        request: pytest.FixtureRequest,
    ) -> None:
        # A short home, since the gpg-agent socket path has a length limit.
        home = Path(tempfile.mkdtemp(dir="/tmp"))
        request.addfinalizer(lambda: shutil.rmtree(home, ignore_errors=True))
        request.addfinalizer(
            lambda: subprocess.run(["gpgconf", "--homedir", str(home), "--kill", "all"])
        )
        gpg = ["gpg", "--homedir", str(home), "--batch", "--passphrase", ""]
        uid = "Report <report@example.com>"
        generate = [*gpg, "--quick-gen-key", uid, "default", "default", "never"]
        subprocess.run(generate, check=True, capture_output=True)
        key = tmp_path / "key.asc"
        key.write_text(
            subprocess.check_output([*gpg, "--armor", "--export"], text=True)
        )
        agent = tmp_path / "private-agent"
        agent.write_text(PRIVATE_AGENT)
        agent.chmod(0o755)
        output = tmp_path / "github_output"
        monkeypatch.setenv("GITHUB_OUTPUT", str(output))
        monkeypatch.setenv("STATE_FILE", str(tmp_path / "state"))
        monkeypatch.setenv("GITHUB_STEP_SUMMARY", str(tmp_path / "summary"))
        runner_temp = tmp_path / "runner"
        runner_temp.mkdir()
        monkeypatch.setenv("RUNNER_TEMP", str(runner_temp))
        login = tmp_path / "claude"
        login.mkdir()
        (login / ".credentials.json").write_text("seat")
        monkeypatch.setenv("CLAUDE_CONFIG_DIR", str(login))
        capfd.readouterr()
        argv = ["--prompt", "Audit", "--agent", "claude", "--bin", str(agent)]
        assert main([*argv, "--output", "report", "--recipient", str(key)]) == 0
        captured = capfd.readouterr()
        assert "secret" not in captured.out + captured.err
        armor = output.read_text().split("\n", 1)[1].rsplit("\nEOF_", 1)[0]
        assert armor.startswith("-----BEGIN PGP MESSAGE-----")
        plain = subprocess.run(
            [*gpg, "--decrypt"], input=armor, capture_output=True, text=True
        ).stdout
        assert plain == "secret report token= summary= login=seat"
        state = Path((tmp_path / "state").read_text().strip())
        assert state.is_relative_to(runner_temp)
        assert not state.exists()
        assert not (tmp_path / "summary").exists()


class TestDelivery:
    """Tests for image storage, timeouts, and Git authentication."""

    def test_should_name_stored_images_uniquely_and_url_safe(self) -> None:
        paths = [Path("before/shot.png"), Path("after/shot.png"), Path("a#1 [x].png")]
        assert stored_names(paths) == ["1-shot.png", "2-shot.png", "3-a_1__x_.png"]

    @pytest.mark.skipif(sys.platform == "win32", reason="uses a POSIX shell")
    def test_should_kill_the_agent_and_its_children_on_timeout(
        self, tmp_path: Path
    ) -> None:
        pid_file = tmp_path / "pid"
        script = f"sleep 60 & echo $! > {pid_file}; wait"
        with pytest.raises(subprocess.TimeoutExpired):
            run_agent(["sh", "-c", script], env={"PATH": "/usr/bin:/bin"}, seconds=1)
        child = pid_file.read_text().strip()
        # The orphaned child may linger as a zombie until init reaps it.
        for _ in range(50):
            ps = subprocess.run(["ps", "-o", "stat=", "-p", child], capture_output=True)
            if not ps.stdout.strip() or ps.stdout.startswith(b"Z"):
                break
            time.sleep(0.1)
        else:
            pytest.fail(f"process {child} still runs")

    def test_should_send_the_app_token_to_git(self) -> None:
        env = git_auth({"GH_TOKEN": "abc"})
        assert env["GIT_CONFIG_KEY_0"] == "http.https://github.com/.extraheader"
        encoded = env["GIT_CONFIG_VALUE_0"].removeprefix("AUTHORIZATION: basic ")
        assert base64.b64decode(encoded) == b"x-access-token:abc"
