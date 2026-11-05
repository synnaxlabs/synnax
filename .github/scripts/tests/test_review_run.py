#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import dataclasses
import json
import os
from pathlib import Path
from typing import Any

import pytest

from review import Config, Doc, Finding, PullRequest, load, run
from review.__main__ import claude as launch

ROOT = Path(__file__).resolve().parents[3]
WORK = Path("/work")
DIFF = """\
diff --git a/core/a.go b/core/a.go
index 1111111..2222222 100644
--- a/core/a.go
+++ b/core/a.go
@@ -10,3 +10,4 @@ func A() {
 keep
-old
+new
+-- not a header
 tail
diff --git a/web/b.ts b/web/b.ts
index 3333333..4444444 100644
--- a/web/b.ts
+++ b/web/b.ts
@@ -1 +1 @@
-x
+y
"""


def config(**kw: Any) -> Config:
    fields: dict[str, Any] = {
        "model": "claude-opus-5-5",
        "effort": "high",
        "tools": ["Read", "Grep", "Glob"],
        "excluded": [],
        "diff_chars": 100_000,
        "caps": {"P0": 2, "P1": 4},
        "prompt": "Review the pull request.",
        "docs": [
            Doc(path="AGENTS.md", description="Rules", text="Every rule."),
            Doc(path="go.md", description="Go", text="Go only.", paths=("**/*.go",)),
            Doc(path="py.md", description="Py", text="Py only.", paths=("**/*.py",)),
        ],
    }
    return Config(**(fields | kw))


def pull(**kw: Any) -> PullRequest:
    fields: dict[str, Any] = {
        "number": 7,
        "title": "Fix A",
        "body": "Fixes A.",
        "head": Path("/head"),
        "diff": DIFF,
        "open": set(),
    }
    return PullRequest(**(fields | kw))


def finding(severity: str) -> dict[str, Any]:
    return {"severity": severity, "title": "T", "body": "B", "path": "a.go", "line": 1}


class FakeClaude:
    """Records the command and prompt of a review, and answers with a fixed result."""

    def __init__(self, score: int = 5, findings: list[dict[str, Any]] | None = None):
        self.args: list[str] = []
        self.stdin = ""
        self.cwd = Path()
        self.out: dict[str, Any] = {
            "is_error": False,
            "subtype": "success",
            "structured_output": {
                "score": score,
                "verdict": "Safe to merge.",
                "summary": ["Fixes A."],
                "findings": findings or [],
            },
        }

    def __call__(self, args: list[str], stdin: str, cwd: Path) -> str:
        self.args, self.stdin, self.cwd = args, stdin, cwd
        return json.dumps(self.out)

    def flag(self, name: str) -> str:
        return self.args[self.args.index(name) + 1]


def review(
    c: Config | None = None, pr: PullRequest | None = None, workdir: Path = WORK
) -> FakeClaude:
    claude = FakeClaude()
    run(c or config(), pr or pull(), claude=claude, workdir=workdir)
    return claude


class TestLoad:
    """Tests loading the configuration in `.sedosaur`."""

    def test_should_load_the_committed_configuration(self) -> None:
        c = load(ROOT)
        assert c.tools == ["Read", "Grep", "Glob"]
        paths = [d.path for d in c.docs]
        assert "AGENTS.md" in paths
        assert ".sedosaur/rules/generated-code.md" in paths
        assert all(d.text.strip() for d in c.docs)

    def test_should_read_the_front_matter_of_a_rule(self, tmp_path: Path) -> None:
        home = tmp_path / ".sedosaur"
        (home / "rules").mkdir(parents=True)
        (home / "config.toml").write_text(
            'model = "m"\neffort = "low"\ntools = []\nexcluded = []\n'
            "diff_chars = 1\ncaps = {}\ncontext = []\n"
        )
        (home / "prompt.md").write_text("Review it.\n")
        (home / "rules" / "go.md").write_text(
            '+++\ndescription = "Go"\npaths = ["**/*.go"]\n+++\n\nNo panics.\n'
        )
        [rule] = load(tmp_path).docs
        assert rule.path == ".sedosaur/rules/go.md"
        assert rule.paths == ("**/*.go",)
        assert rule.text == "\nNo panics.\n"

    def test_should_reject_an_unknown_key(self, tmp_path: Path) -> None:
        home = tmp_path / ".sedosaur"
        (home / "rules").mkdir(parents=True)
        (home / "config.toml").write_text("context = []\ncolour = 1\n")
        (home / "prompt.md").write_text("Review it.\n")
        with pytest.raises(TypeError, match="unexpected keyword argument 'colour'"):
            load(tmp_path)


class TestDocs:
    """Tests which documents apply to a pull request."""

    @pytest.mark.parametrize(
        ("glob", "path", "applies"),
        [
            ("**/*.go", "a.go", True),
            ("**/*.go", "core/pkg/a.go", True),
            ("*.go", "core/a.go", False),
            ("core/**", "core/pkg/a.go", True),
            ("**/*.ts", "a.tsx", False),
        ],
    )
    def test_should_match_paths_by_glob(
        self, glob: str, path: str, applies: bool
    ) -> None:
        doc = Doc(path="d.md", description="D", text="T", paths=(glob,))
        assert doc.applies([path]) is applies

    def test_should_put_only_the_documents_that_apply_in_the_prompt(self) -> None:
        stdin = review().stdin
        assert "## `AGENTS.md`: Rules\n\nEvery rule." in stdin
        assert "Go only." in stdin
        assert "Py only." not in stdin


class TestPrompt:
    """Tests the prompt that a review sends to Claude Code."""

    def test_should_number_the_lines_the_head_keeps(self) -> None:
        stdin = review().stdin
        numbered = "    10  keep\n       -old\n    11 +new\n    12 +-- not a header"
        assert numbered in stdin
        assert "    13  tail" in stdin

    def test_should_name_the_pull_request_and_its_checkout(self) -> None:
        stdin = review().stdin
        assert stdin.startswith("Review the pull request.\n\n# Pull request #7: Fix A")
        assert "The repository at the head of the pull request is in `/head`." in stdin

    def test_should_list_the_open_findings(self) -> None:
        stdin = review(pr=pull(open={("core/a.go", "Old bug")})).stdin
        assert "# Earlier findings still open\n\n- `core/a.go`: Old bug" in stdin

    def test_should_cut_the_diff_at_a_file(self, tmp_path: Path) -> None:
        claude = review(config(diff_chars=200), workdir=tmp_path)
        assert claude.cwd == tmp_path
        assert "+new" in claude.stdin
        assert "+y" not in claude.stdin
        assert "in `rest.diff` in your working directory" in claude.stdin
        assert "Read all of it:\n- `web/b.ts`" in claude.stdin
        rest = (tmp_path / "rest.diff").read_text()
        assert rest.startswith("diff --git a/web/b.ts b/web/b.ts")
        assert "       -x\n     1 +y" in rest

    def test_should_write_no_file_when_the_diff_fits(self, tmp_path: Path) -> None:
        review(workdir=tmp_path)
        assert list(tmp_path.iterdir()) == []


class TestCommand:
    """Tests the command that runs Claude Code."""

    def test_should_confine_claude_to_reading_the_head(self) -> None:
        claude = review()
        assert claude.args[:3] == ["claude", "--print", "--restricted"]
        assert claude.flag("--tools") == "Read,Grep,Glob"
        assert claude.flag("--permission-mode") == "dontAsk"
        assert claude.flag("--add-dir") == "/head"
        assert "--strict-mcp-config" in claude.args
        assert claude.flag("--model") == "claude-opus-5-5"

    def test_should_ask_for_every_field_of_a_finding(self) -> None:
        schema = json.loads(review().flag("--json-schema"))
        props = schema["properties"]["findings"]["items"]["properties"]
        assert set(props) == {f.name for f in dataclasses.fields(Finding)}


class TestScore:
    """Tests the score a review gives."""

    @pytest.mark.parametrize(
        ("severities", "score"),
        [([], 5), (["P2"], 5), (["P2", "P1"], 4), (["P1", "P0"], 2)],
    )
    def test_should_cap_it_by_the_most_severe_finding(
        self, severities: list[str], score: int
    ) -> None:
        claude = FakeClaude(5, [finding(s) for s in severities])
        assert run(config(), pull(), claude=claude, workdir=WORK).score == score

    def test_should_keep_a_score_below_the_cap(self) -> None:
        claude = FakeClaude(3, [finding("P1")])
        assert run(config(), pull(), claude=claude, workdir=WORK).score == 3

    def test_should_raise_when_claude_ends_with_an_error(self) -> None:
        claude = FakeClaude()
        claude.out = {"is_error": True, "subtype": "error_max_turns", "result": "Out."}
        with pytest.raises(
            RuntimeError, match=r"claude ended with error_max_turns: Out\."
        ):
            run(config(), pull(), claude=claude, workdir=WORK)


class TestLaunch:
    """Tests the process that runs Claude Code."""

    def test_should_pass_claude_only_its_token(
        self, tmp_path: Path, monkeypatch: pytest.MonkeyPatch
    ) -> None:
        bindir = tmp_path / "bin"
        bindir.mkdir()
        fake = bindir / "claude"
        fake.write_text(
            "#!/usr/bin/env python3\n"
            "import json, os, sys\n"
            "print(json.dumps({'args': sys.argv[1:], 'stdin': sys.stdin.read(),"
            " 'cwd': os.getcwd(), 'env': dict(os.environ)}))\n"
        )
        fake.chmod(0o755)
        monkeypatch.setenv("PATH", f"{bindir}{os.pathsep}{os.environ['PATH']}")
        monkeypatch.setenv("GH_TOKEN", "app-token")
        monkeypatch.setenv("CLAUDE_CODE_OAUTH_TOKEN", "claude-token")
        work = tmp_path / "work"
        work.mkdir()
        out = json.loads(launch(["claude", "--print"], "Review it.", work))
        assert out["args"] == ["--print"]
        assert out["stdin"] == "Review it."
        assert Path(out["cwd"]).resolve() == work.resolve()
        env = out["env"]
        assert env["CLAUDE_CODE_OAUTH_TOKEN"] == "claude-token"
        assert "GH_TOKEN" not in env
        assert env["HOME"] == env["CLAUDE_CONFIG_DIR"] != os.environ["HOME"]
