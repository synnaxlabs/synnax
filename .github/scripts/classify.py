#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Labels a pull request review/human or review/bot and explains why in one comment.

    python3 classify.py <results-dir>

A pull request needs review/human when a surface report differs between its base and
head, a report is missing, or a trust check fails. A surface the head predates reports
skipped, which does not raise the tier. Otherwise the author picks the tier, so the
script adds no label. It lowers review/human only when nothing flags the pull request
now and the label is still the one it added, which restores the tier from before. It
reads the pull request number from PR, or else <results-dir>/pr/number, and refuses it
unless the pull request's head is HEAD_SHA. When a pull request that a bot opened needs
review/human, it requests a review from REVIEWER, if set, since no author will. It asks
only while nobody was ever asked to review the pull request or reviewed it.
It reads only its own comment: one that the App named APP posted.
Environment: APP, GH_TOKEN, GITHUB_REPOSITORY, HEAD_SHA, and optionally PR and REVIEWER.
"""

import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import Any
from urllib.parse import quote

HUMAN = "review/human"
BOT = "review/bot"
# Tier labels from before the two-tier scheme. Both ask for a human.
HUMAN_TIERS = {HUMAN, "review/light", "review/thorough"}
# Files that make or judge the reports. A pull request that changes one could forge
# its own result, so it always gets a human.
CLASSIFIER_PATHS = (
    ".github/scripts/classify.py",
    ".github/scripts/surface.sh",
    ".github/workflows/check.classify.yaml",
    ".github/workflows/label.classify.yaml",
    ".github/actions/",
    "client/ts/scripts/surface.sh",
    "console/scripts/surface",
    "core/cmd/surface/",
    "oracle/cmd/surface.go",
    "oracle/surface/",
    "scripts/surface_cpp.py",
    "scripts/surface_driver.py",
    "scripts/surface_py.py",
    "scripts/surface_py_cli.py",
    "scripts/surface_py_state.py",
)
MARKER = "<!-- review-classifier -->"
NO_TIER = "none"
SURFACES = {
    "oracle": "Oracle schemas",
    "client-ts": "TypeScript client API",
    "console-state": "Console saved state",
    "client-py": "Python client API",
    "client-py-cli": "Python client CLI",
    "client-py-state": "Python client saved state",
    "core": "Core flags and config keys",
    "core-windows": "Core commands and flags on Windows",
    "driver": "Driver commands and config keys",
    "client-cpp": "C++ client API",
}
STATUS_TEXT = {
    "same": "No change",
    "changed": "Changed",
    "error": "Report failed",
    "skipped": "Not compared: the branch predates this report",
}
FLAGGED = {"changed", "error"}
MAX_DIFF_CHARS = 12_000


@dataclass(kw_only=True)
class Surface:
    name: str
    status: str
    diff: str


@dataclass(kw_only=True, frozen=True)
class Raise:
    """A review/human label the classifier added: the tier before it, and how many times
    review/human had been added to the pull request once it was on."""

    tier: str
    adds: int

    def line(self) -> str:
        return f"<!-- review-classifier-raised-from: {self.tier} #{self.adds} -->"


RAISE_LINE = re.compile(
    r"<!-- review-classifier-raised-from: (review/bot|none) #(\d+) -->"
)


def read_raise(body: str) -> Raise | None:
    """Reads the raise that render wrote on the line after MARKER."""
    lines = body.splitlines()
    match = RAISE_LINE.fullmatch(lines[1]) if len(lines) > 1 else None
    return Raise(tier=match.group(1), adds=int(match.group(2))) if match else None


def read_surfaces(root: Path) -> list[Surface]:
    """Reads the uploaded results and the run's job names from <root>/jobs. A run older
    than a surface has no job for it, which is not a failure. A run with no surface job
    at all never compared anything, so each of its reports is an error."""
    jobs = (root / "jobs").read_text().splitlines()
    compared = {name for name in SURFACES if f"Compare {name}" in jobs}
    out = []
    for name in SURFACES:
        d = root / f"surface-{name}"
        status = (d / "status").read_text().strip() if (d / "status").exists() else ""
        diff = (d / "diff").read_text() if (d / "diff").exists() else ""
        if status not in STATUS_TEXT and compared and name not in compared:
            status, diff = "skipped", ""
        elif status not in STATUS_TEXT:
            status, diff = "error", "No report: the job did not finish.\n"
        out.append(Surface(name=name, status=status, diff=diff))
    return out


def trust_reasons(pull: dict[str, Any], *, files: list[dict[str, Any]]) -> list[str]:
    reasons = []
    if pull["head"]["repo"]["full_name"] != pull["base"]["repo"]["full_name"]:
        reasons.append("The pull request comes from a fork.")
    changed = sorted(
        f["filename"] for f in files if f["filename"].startswith(CLASSIFIER_PATHS)
    )
    if changed:
        reasons.append(
            f"The pull request changes the classifier: {', '.join(changed)}."
        )
    return reasons


def needs_human(surfaces: list[Surface], *, reasons: list[str]) -> bool:
    return any(s.status in FLAGGED for s in surfaces) or bool(reasons)


def plan_labels(
    current: set[str], *, human: bool, raised: Raise | None, adds: int
) -> tuple[list[str], list[str], Raise | None]:
    """Returns the labels to add and to remove, and the raise to record. Raises to
    review/human when needed and otherwise leaves the author's tier alone. An existing
    human tier counts as review/human. raised is the recorded raise, and adds is how
    many times review/human was added. The raise still holds only while adds matches
    it, so a label anyone removed and added again stays."""
    bot = [BOT] if BOT in current else []
    on = raised is not None and raised.adds == adds and HUMAN in current
    held = raised if on else None
    if human:
        if current & HUMAN_TIERS:
            return ([] if HUMAN in current else [HUMAN]), bot, held
        return [HUMAN], bot, Raise(tier=BOT if bot else NO_TIER, adds=adds + 1)
    if held is not None:
        return ([BOT] if held.tier == BOT and not bot else []), [HUMAN], None
    if current & HUMAN_TIERS:
        return ([] if HUMAN in current else [HUMAN]), bot, None
    return [], [], None


def fence(text: str) -> str:
    longest = max((len(m) for m in re.findall(r"`+", text)), default=0)
    ticks = "`" * max(3, longest + 1)
    if len(text) > MAX_DIFF_CHARS:
        cut = text[:MAX_DIFF_CHARS]
        text = cut + f"\n... {text.count(chr(10)) - cut.count(chr(10))} more lines\n"
    return f"{ticks}diff\n{text.rstrip()}\n{ticks}"


def render(
    surfaces: list[Surface],
    *,
    reasons: list[str],
    kept: bool,
    raised: Raise | None,
    lowered: bool,
) -> str:
    changed = any(s.status in FLAGGED for s in surfaces)
    tier = f"`{HUMAN}`" if changed or reasons or kept else "the author picks"
    if lowered:
        summary = f"No report is flagged now, so the classifier removed its `{HUMAN}`."
    elif changed:
        summary = "This pull request changes a user-facing interface."
    elif reasons:
        summary = "The classifier cannot clear this pull request."
    elif kept:
        summary = "No interface changed, but a human review tier is already set."
    else:
        summary = "No report changed, so the classifier sets no tier."
    rows = [f"| {SURFACES[s.name]} | {STATUS_TEXT[s.status]} |" for s in surfaces]
    lines = [
        MARKER,
        *([raised.line()] if raised is not None else []),
        f"## Review tier: {tier}",
        "",
        summary,
        "",
        *[f"- {r}" for r in reasons],
        *([""] if reasons else []),
        "| Interface | Result |",
        "| --- | --- |",
        *rows,
    ]
    for s in surfaces:
        if s.status not in FLAGGED:
            continue
        lines += [
            "",
            f"<details><summary>{SURFACES[s.name]}</summary>",
            "",
            fence(s.diff),
            "",
            "</details>",
        ]
    lines += [
        "",
        f"The classifier removes `{HUMAN}` only when it added the label itself.",
        "To keep the tier, remove the label and add it again.",
    ]
    return "\n".join(lines) + "\n"


def gh(*args: str, stdin: str | None = None) -> str:
    return subprocess.run(
        ["gh", *args], input=stdin, capture_output=True, text=True, check=True
    ).stdout


def gh_lines(*args: str) -> list[Any]:
    return [json.loads(line) for line in gh(*args).splitlines() if line]


def find_comment(repo: str, *, number: int, app: str) -> dict[str, Any] | None:
    comments = gh_lines(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/comments",
        "--jq",
        ".[] | {id, body, app: .performed_via_github_app.slug}",
    )
    return next(
        (c for c in comments if c["app"] == app and c["body"].startswith(MARKER)),
        None,
    )


def write_comment(
    repo: str, *, number: int, comment: dict[str, Any] | None, body: str
) -> None:
    if comment is not None and comment["body"] == body:
        return
    target = (
        f"repos/{repo}/issues/{number}/comments"
        if comment is None
        else f"repos/{repo}/issues/comments/{comment['id']}"
    )
    gh(
        "api",
        "-X",
        "POST" if comment is None else "PATCH",
        target,
        "--input",
        "-",
        stdin=json.dumps({"body": body}),
    )


def count_human_adds(repo: str, *, number: int) -> int:
    """Returns how many times review/human was added to the pull request."""
    events = gh(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/timeline",
        "--jq",
        f'.[] | select(.event == "labeled" and .label.name == "{HUMAN}") | .event',
    )
    return len(events.split())


def reviewers_engaged(repo: str, *, number: int) -> bool:
    """Returns whether anyone was ever asked to review the pull request, or a person
    reviewed it. A bot's review, such as Sedosaur's, does not count."""
    events = gh(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/timeline",
        "--jq",
        '.[] | select(.event == "review_requested"'
        ' or (.event == "reviewed" and .user.type != "Bot")) | .event',
    )
    return bool(events.strip())


def set_labels(repo: str, *, number: int, add: list[str], remove: list[str]) -> None:
    for label in add:
        gh(
            "api",
            "-X",
            "POST",
            f"repos/{repo}/issues/{number}/labels",
            "-f",
            f"labels[]={label}",
        )
    for label in remove:
        gh(
            "api",
            "-X",
            "DELETE",
            f"repos/{repo}/issues/{number}/labels/{quote(label, safe='')}",
        )


def main() -> None:
    root = Path(sys.argv[1])
    repo = os.environ["GITHUB_REPOSITORY"]
    head_sha = os.environ["HEAD_SHA"]
    number = int(os.environ.get("PR") or (root / "pr" / "number").read_text().strip())
    pull = json.loads(gh("api", f"repos/{repo}/pulls/{number}"))
    if pull["head"]["sha"] != head_sha:
        print(f"#{number} moved past {head_sha}; a newer run classifies it")
        return
    files = gh_lines(
        "api",
        "--paginate",
        f"repos/{repo}/pulls/{number}/files",
        "--jq",
        ".[] | {filename}",
    )
    comment = find_comment(repo, number=number, app=os.environ["APP"])
    reasons = trust_reasons(pull, files=files)
    surfaces = read_surfaces(root)
    # A push during the reads leaves the reports stale: check the head before writing.
    pull = json.loads(gh("api", f"repos/{repo}/pulls/{number}"))
    if pull["head"]["sha"] != head_sha:
        print(f"#{number} moved past {head_sha}; a newer run classifies it")
        return
    current = {label["name"] for label in pull["labels"]}
    raised = read_raise(comment["body"]) if comment is not None else None
    add, remove, raised = plan_labels(
        current,
        human=needs_human(surfaces, reasons=reasons),
        raised=raised,
        adds=count_human_adds(repo, number=number),
    )
    kept = bool(current & HUMAN_TIERS) and HUMAN not in remove
    body = render(
        surfaces, reasons=reasons, kept=kept, raised=raised, lowered=HUMAN in remove
    )
    # A record whose label write fails never matches the count, so it goes first.
    if HUMAN in add:
        write_comment(repo, number=number, comment=comment, body=body)
    set_labels(repo, number=number, add=add, remove=remove)
    if HUMAN not in add:
        write_comment(repo, number=number, comment=comment, body=body)
    reviewer = os.environ.get("REVIEWER", "").strip()
    human = HUMAN in add or HUMAN in current
    bot = pull.get("user", {}).get("type") == "Bot"
    if human and reviewer and bot and not reviewers_engaged(repo, number=number):
        gh(
            "api",
            "-X",
            "POST",
            f"repos/{repo}/pulls/{number}/requested_reviewers",
            "-f",
            f"reviewers[]={reviewer}",
        )
    print(f"#{number}: add {add}, remove {remove}")


if __name__ == "__main__":
    main()
