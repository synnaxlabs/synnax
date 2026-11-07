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
from concurrent.futures import ThreadPoolExecutor
from dataclasses import dataclass, replace
from pathlib import Path
from typing import Any

from review.findings import SCHEMA, SEVERITIES, Finding, Review, parse
from review.render import fence

Claude = Callable[[list[str], str, Path], str]
"""Runs `claude` with the given arguments and stdin in a working directory, and returns
its stdout."""

REST = "rest.diff"
"""The file in the working directory that holds the diff the prompt leaves out."""
NEW = "new.diff"
"""The file in the working directory that holds the new commits when they are too long
to show."""

_HUNK = re.compile(r"@@ -\d+(?:,\d+)? \+(\d+)")
_FILE = re.compile(r"^diff --git a/.* b/(.*)$", re.M)
_GLOB = {"**/": "(?:.*/)?", "**": ".*", "*": "[^/]*", "?": "[^/]"}
# Reviewers write these blocks into the body. A review must not see another review.
_REVIEWS = re.compile(
    r"<!-- (greptile|sedosaur)_comment -->.*?<!-- /\1_comment -->", re.S
)


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
class Lens:
    """A focus that one pass of a review takes."""

    name: str
    focus: str
    """What the pass looks for, written to the reviewer."""
    effort: str | None = None
    """The effort of the pass, if not the effort of the config."""


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
    lenses: list[Lens]
    """The focus of each pass. With none, one pass reviews the pull request. With
    some, one pass runs for each in parallel, and the review joins their findings."""


@dataclass(frozen=True, kw_only=True)
class Window:
    """A usage limit window of the Claude subscription, such as `seven_day`."""

    status: str
    utilization: float
    """The share of the window's limit in use, from 0 to 1."""
    resets_at: int
    """When the window resets, in Unix seconds."""


@dataclass(frozen=True, kw_only=True)
class Usage:
    """What one review cost, as Claude Code reports it."""

    cost_usd: float
    duration_ms: int
    turns: int
    input_tokens: int
    output_tokens: int
    cache_read_tokens: int
    cache_creation_tokens: int
    limits: dict[str, Window]
    """The subscription's limit windows after the review, by name. Empty when Claude
    Code reports none, as with an API key."""

    @classmethod
    def combine(cls, passes: list["Usage"]) -> "Usage":
        """Adds up the usage of passes that ran in parallel. The duration is the
        longest pass, and the limits are those of the last pass to end that reported
        any."""
        return cls(
            cost_usd=sum(u.cost_usd for u in passes),
            duration_ms=max(u.duration_ms for u in passes),
            turns=sum(u.turns for u in passes),
            input_tokens=sum(u.input_tokens for u in passes),
            output_tokens=sum(u.output_tokens for u in passes),
            cache_read_tokens=sum(u.cache_read_tokens for u in passes),
            cache_creation_tokens=sum(u.cache_creation_tokens for u in passes),
            limits=max(
                (u for u in passes if u.limits),
                key=lambda u: u.duration_ms,
                default=passes[0],
            ).limits,
        )

    @classmethod
    def of(cls, out: dict[str, Any], messages: list[dict[str, Any]]) -> "Usage":
        """Reads the usage from Claude Code's result and the rate limit events among
        the messages it streamed."""
        u = out["usage"]
        events = [
            m["rate_limit_info"] for m in messages if m["type"] == "rate_limit_event"
        ]
        windows = events[-1]["unifiedWindows"] if events else {}
        return cls(
            cost_usd=out["total_cost_usd"],
            duration_ms=out["duration_ms"],
            turns=out["num_turns"],
            input_tokens=u["input_tokens"],
            output_tokens=u["output_tokens"],
            cache_read_tokens=u["cache_read_input_tokens"],
            cache_creation_tokens=u["cache_creation_input_tokens"],
            limits={
                name: Window(
                    status=w["status"],
                    utilization=w["utilization"],
                    resets_at=w["resetsAt"],
                )
                for name, w in windows.items()
            },
        )


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
    new: str = ""
    """The patches of the pull request's own commits after the last review, if any."""


def load(root: Path) -> Config:
    """Loads the configuration in `.sedosaur`, with the documents it names.

    :param root: The root of a trusted checkout of the default branch.
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
    lenses = [Lens(**lens) for lens in data.pop("lens", [])]
    return Config(
        **data, prompt=(home / "prompt.md").read_text(), docs=docs, lenses=lenses
    )


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


def _cut(diff: str, chars: int) -> tuple[str, str]:
    """Cuts a diff at the last file boundary within chars, and returns both parts."""
    size = 0
    files = re.split(r"(?m)^(?=diff --git )", diff)
    for i, chunk in enumerate(files):
        if size + len(chunk) > chars:
            return "".join(files[:i]), "".join(files[i:])
        size += len(chunk)
    return diff, ""


def _prompt(config: Config, pr: PullRequest, shown: str, rest: str) -> str:
    """Builds the reviewer's prompt for a pull request."""
    changed = _FILE.findall(pr.diff)
    parts = [
        config.prompt.strip(),
        f"# Pull request #{pr.number}: {pr.title}",
        _REVIEWS.sub("", pr.body).strip() or "No description.",
    ]
    if pr.open:
        parts += [
            "# Earlier findings still open",
            "\n".join(f"- `{path}`: {title}" for path, title in sorted(pr.open)),
        ]
    if pr.new:
        parts += [
            "# Commits since your last review",
            "Review these changes first, as new code: a fix for an earlier finding can "
            "add a bug. Anchor findings to the line numbers of the full diff.",
        ]
        if _shows_new(config, pr):
            parts.append(fence(pr.new, "diff"))
        else:
            parts.append(f"They are too long to show. Read all of `{NEW}` first.")
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
    if rest:
        listed = "\n".join(f"- `{p}`" for p in _FILE.findall(rest))
        parts.append(
            "The diff is too long to show in full. The diff of these files is in "
            f"`{REST}` in your working directory. Read all of it:\n{listed}"
        )
    parts.append(fence(shown, "diff"))
    return "\n\n".join(parts)


def _shows_new(config: Config, pr: PullRequest) -> bool:
    """Reports whether the new commits fit in the prompt."""
    return len(pr.new) <= config.diff_chars // 3


def _command(config: Config, *, head: Path, effort: str | None = None) -> list[str]:
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
        effort or config.effort,
        "--output-format",
        "stream-json",
        "--verbose",
        "--json-schema",
        json.dumps(SCHEMA),
        "--no-session-persistence",
    ]


def _cap(review: Review, caps: dict[str, int]) -> Review:
    """Lowers the score of a review to the cap of its most severe finding."""
    limit = min((caps.get(f.severity, 5) for f in review.findings), default=5)
    return replace(review, score=min(review.score, limit))


def _focused(text: str, lens: Lens) -> str:
    """Builds the prompt of one pass from the review prompt and the pass's lens."""
    return (
        f"{text}\n\n# Your focus: {lens.name}\n\n{lens.focus.strip()}\n\n"
        "Spend your review on this focus. Report any other defect you confirm too."
    )


def _merge(
    reviews: list[Review], *, caps: dict[str, int], open: set[tuple[str, str]]
) -> Review:
    """Joins the reviews of parallel passes. Findings of two passes in one file that
    end on the same line or have the same title are one finding: the review keeps the
    more severe, else the one that continues an open thread. The verdict and summary
    come from the pass with the lowest capped score."""

    def rank(f: Finding) -> tuple[int, bool]:
        return SEVERITIES.index(f.severity), (f.path, f.title) not in open

    def same(f: Finding, k: Finding) -> bool:
        return f.path == k.path and (f.line == k.line or f.title == k.title)

    kept: list[tuple[int, Finding]] = []
    every = [(i, f) for i, r in enumerate(reviews) for f in r.findings]
    for i, f in sorted(every, key=lambda t: rank(t[1])):
        if not any(j != i and same(f, k) for j, k in kept):
            kept.append((i, f))
    lead = min(reviews, key=lambda r: _cap(r, caps).score)
    return replace(
        lead, score=min(r.score for r in reviews), findings=[f for _, f in kept]
    )


def _pass(
    config: Config,
    pr: PullRequest,
    text: str,
    *,
    claude: Claude,
    workdir: Path,
    effort: str | None = None,
) -> tuple[Review, Usage]:
    stream = claude(_command(config, head=pr.head, effort=effort), text, workdir)
    messages = [json.loads(line) for line in stream.splitlines() if line]
    [out] = [m for m in messages if m["type"] == "result"]
    if out["is_error"]:
        raise RuntimeError(f"claude ended with {out['subtype']}: {out.get('result')}")
    return parse(out["structured_output"]), Usage.of(out, messages)


def run(
    config: Config, pr: PullRequest, *, claude: Claude, workdir: Path
) -> tuple[Review, Usage]:
    """Reviews a pull request, and returns the review with what it cost.

    :param claude: Called from one thread for each lens when the config has lenses.
    :param workdir: An empty directory that Claude Code starts in and can read.
    :raises RuntimeError: If Claude Code ends with an error.
    """
    shown, rest = _cut(_numbered(pr.diff), config.diff_chars)
    if rest:
        (workdir / REST).write_text(rest)
    if not _shows_new(config, pr):
        (workdir / NEW).write_text(pr.new)
    text = _prompt(config, pr, shown, rest)
    if not config.lenses:
        review, usage = _pass(config, pr, text, claude=claude, workdir=workdir)
        return _cap(review, config.caps), usage
    with ThreadPoolExecutor(len(config.lenses)) as pool:
        passes = list(
            pool.map(
                lambda lens: _pass(
                    config,
                    pr,
                    _focused(text, lens),
                    claude=claude,
                    workdir=workdir,
                    effort=lens.effort,
                ),
                config.lenses,
            )
        )
    review = _merge([r for r, _ in passes], caps=config.caps, open=pr.open)
    return _cap(review, config.caps), Usage.combine([u for _, u in passes])
