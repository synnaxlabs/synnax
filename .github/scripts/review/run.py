#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

import json
import re
import tomllib
from collections.abc import Callable, Sequence
from dataclasses import dataclass, replace
from pathlib import Path

from review.findings import SCHEMA, Review, parse
from review.render import fence

Claude = Callable[[list[str], str], str]
"""Runs `claude` with the given arguments and stdin, and returns its stdout."""

_HUNK = re.compile(r"@@ -\d+(?:,\d+)? \+(\d+)")
_FILE = re.compile(r"^diff --git a/.* b/(.*)$", re.M)
_GLOB = {"**/": "(?:.*/)?", "**": ".*", "*": "[^/]*", "?": "[^/]"}


def _matches(path: str, glob: str) -> bool:
    """Reports whether a repository path matches a glob. `**` spans directories."""
    parts = re.split(r"(\*\*/|\*\*|\*|\?)", glob)
    return (
        re.fullmatch("".join(_GLOB.get(p, re.escape(p)) for p in parts), path)
        is not None
    )


@dataclass(frozen=True, kw_only=True)
class Doc:
    """A document whose rules the reviewer applies."""

    path: str
    description: str
    text: str
    paths: tuple[str, ...] = ()
    """Globs of changed files that make the document apply. Empty applies always."""

    def applies(self, changed: Sequence[str]) -> bool:
        return not self.paths or any(
            _matches(c, g) for c in changed for g in self.paths
        )


@dataclass(frozen=True, kw_only=True)
class Config:
    """How the reviewer reviews, from `.sedosaur/config.toml`."""

    model: str
    effort: str
    tools: list[str]
    excluded: list[str]
    """Git globs of files left out of the diff."""
    diff_chars: int
    caps: dict[str, int]
    """The highest score a review can give while it has a finding of the severity."""
    prompt: str
    docs: list[Doc]


@dataclass(frozen=True, kw_only=True)
class PullRequest:
    number: int
    title: str
    body: str
    head: Path
    """A checkout of the pull request's head commit."""
    diff: str
    """The diff from the merge base to the head."""
    open: set[tuple[str, str]]
    """The path and title of each finding of an earlier review still open."""


def load(root: Path) -> Config:
    """Loads the configuration in `.sedosaur`, with the documents it names.

    :param root: The root of a checkout of the base branch.
    :raises TypeError: If the config has a missing or unknown key.
    """
    home = root / ".sedosaur"
    data = tomllib.loads((home / "config.toml").read_text())
    docs = [
        Doc(
            path=c["path"],
            description=c["description"],
            text=(root / c["path"]).read_text(),
            paths=tuple(c.get("paths", ())),
        )
        for c in data.pop("context")
    ]
    for rule in sorted((home / "rules").glob("*.md")):
        _, front, text = rule.read_text().split("+++\n", 2)
        meta = tomllib.loads(front)
        docs.append(
            Doc(
                path=str(rule.relative_to(root)),
                description=meta["description"],
                text=text,
                paths=tuple(meta.get("paths", ())),
            )
        )
    return Config(**data, prompt=(home / "prompt.md").read_text(), docs=docs)


def _numbered(diff: str) -> str:
    """Prefixes each diff line that the head keeps with its line number in the head."""
    out: list[str] = []
    n = 0
    hunk = False
    for line in diff.splitlines():
        if line.startswith("diff --git "):
            hunk = False
        elif m := _HUNK.match(line):
            hunk, n = True, int(m[1])
        elif hunk and line.startswith(("+", " ")):
            out.append(f"{n:>6} {line}")
            n += 1
            continue
        elif hunk and line.startswith("-"):
            out.append(f"{'':>6} {line}")
            continue
        out.append(line)
    return "\n".join(out)


def _cut(diff: str, chars: int) -> tuple[str, list[str]]:
    """Cuts a diff at the last file boundary within chars.

    :returns: The kept diff, and the paths of the files it leaves out.
    """
    kept: list[str] = []
    size = 0
    files = re.split(r"(?m)^(?=diff --git )", diff)
    for i, chunk in enumerate(files):
        if size + len(chunk) > chars:
            return "".join(kept), _FILE.findall("".join(files[i:]))
        kept.append(chunk)
        size += len(chunk)
    return diff, []


def _prompt(config: Config, pr: PullRequest) -> str:
    """Builds the reviewer's prompt for a pull request."""
    changed = _FILE.findall(pr.diff)
    shown, left = _cut(_numbered(pr.diff), config.diff_chars)
    parts = [
        config.prompt.strip(),
        f"# Pull request #{pr.number}: {pr.title}",
        pr.body.strip() or "No description.",
    ]
    if pr.open:
        parts += [
            "# Earlier findings still open",
            "\n".join(f"- `{path}`: {title}" for path, title in sorted(pr.open)),
        ]
    parts.append("# Rules")
    parts += [
        f"## `{d.path}`: {d.description}\n\n{d.text.strip()}"
        for d in config.docs
        if d.applies(changed)
    ]
    parts += [
        "# Diff",
        f"The repository at the head of the pull request is in `{pr.head}`.",
    ]
    if left:
        listed = "\n".join(f"- `{p}`" for p in left)
        parts.append(
            f"The diff is too long to show in full. Read these files:\n{listed}"
        )
    parts.append(fence(shown, "diff"))
    return "\n\n".join(parts)


def _command(config: Config, *, head: Path) -> list[str]:
    """Returns the `claude` command that reviews the code checked out at head.

    Restricted mode confines the tools to the working directory and head, and ignores
    every settings file.
    """
    return [
        "claude",
        "--print",
        "--restricted",
        "--tools",
        ",".join(config.tools),
        "--strict-mcp-config",
        "--permission-mode",
        "dontAsk",
        "--add-dir",
        str(head),
        "--model",
        config.model,
        "--effort",
        config.effort,
        "--output-format",
        "json",
        "--json-schema",
        json.dumps(SCHEMA),
        "--no-session-persistence",
    ]


def _cap(review: Review, caps: dict[str, int]) -> Review:
    """Lowers the score of a review to the cap of its most severe finding."""
    limit = min((caps.get(f.severity, 5) for f in review.findings), default=5)
    return replace(review, score=min(review.score, limit))


def run(config: Config, pr: PullRequest, *, claude: Claude) -> Review:
    """Reviews a pull request.

    :raises RuntimeError: If Claude Code ends with an error.
    """
    out = json.loads(claude(_command(config, head=pr.head), _prompt(config, pr)))
    if out["is_error"]:
        raise RuntimeError(f"claude ended with {out['subtype']}: {out.get('result')}")
    return _cap(parse(out["structured_output"]), config.caps)
