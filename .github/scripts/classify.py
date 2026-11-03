#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Labels a pull request review/human or review/bot and explains why in one comment.

    python3 classify.py <artifacts-dir>

A pull request needs review/human when a surface report differs between its base and
head. No report covers the Console, so the author picks the tier of a UI change. A UI
change with no tier, or with a review/bot label this script added, gets review/human.
The script only raises: it never removes review/human. It reads the pull request
number from <artifacts-dir>/pr/number and refuses it unless the pull request's head
is HEAD_SHA. Environment: GH_TOKEN, GITHUB_REPOSITORY, and HEAD_SHA.
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
APP = "synnax-claude-code[bot]"
# Tier labels from before the two-tier scheme. Both ask for a human.
HUMAN_TIERS = {HUMAN, "review/light", "review/thorough"}
UI_PREFIXES = ("console/", "pluto/", "lyra/")
# Files that make or judge the reports. A pull request that changes one could forge
# its own result, so it always gets a human.
CLASSIFIER_PATHS = (
    ".github/scripts/classify.py",
    ".github/scripts/surface.sh",
    ".github/workflows/check.classify.yaml",
    ".github/workflows/label.classify.yaml",
    ".github/actions/",
    "client/ts/scripts/surface.sh",
    "core/cmd/surface/",
    "oracle/cmd/surface.go",
    "oracle/surface/",
    "scripts/surface_driver.py",
    "scripts/surface_py.py",
)
MARKER = "<!-- review-classifier -->"
# Records in the comment that this script added review/bot: the author did not pick it.
AUTO_BOT = "<!-- review-classifier:auto-bot -->"
SURFACES = {
    "oracle": "Oracle schemas",
    "client-ts": "TypeScript client API",
    "client-py": "Python client API",
    "core": "Core flags and config keys",
    "driver": "Driver commands and config keys",
}
STATUS_TEXT = {"same": "No change", "changed": "Changed", "error": "Report failed"}
MAX_DIFF_CHARS = 12_000


@dataclass(kw_only=True)
class Surface:
    name: str
    status: str
    diff: str


def read_surfaces(root: Path) -> list[Surface]:
    out = []
    for name in SURFACES:
        d = root / f"surface-{name}"
        status = (d / "status").read_text().strip() if (d / "status").exists() else ""
        diff = (d / "diff").read_text() if (d / "diff").exists() else ""
        if status not in STATUS_TEXT:
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


def changes_ui(files: list[dict[str, Any]]) -> bool:
    return any(f["filename"].startswith(UI_PREFIXES) for f in files)


def has_chosen_tier(current: set[str], *, auto_bot: bool) -> bool:
    if current & HUMAN_TIERS:
        return True
    return BOT in current and not auto_bot


def needs_human(surfaces: list[Surface], *, reasons: list[str]) -> bool:
    return any(s.status != "same" for s in surfaces) or bool(reasons)


def plan_labels(current: set[str], *, human: bool) -> tuple[list[str], list[str]]:
    """Returns the labels to add and to remove, so exactly one of review/human and
    review/bot stays. An existing human tier label counts as review/human."""
    if human or current & HUMAN_TIERS:
        return ([] if HUMAN in current else [HUMAN]), ([BOT] if BOT in current else [])
    return ([] if BOT in current else [BOT]), []


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
    ui: bool,
    reasons: list[str],
    kept: bool,
    auto_bot: bool,
) -> str:
    changed = any(s.status != "same" for s in surfaces)
    tier = HUMAN if changed or reasons or kept else BOT
    if changed:
        summary = "This pull request changes a user-facing interface."
    elif reasons:
        summary = "The classifier cannot clear this pull request."
    elif kept:
        summary = "No interface changed, but a human review tier is already set."
    elif ui:
        summary = "No report changed, and the author chose this tier for a UI change."
    else:
        summary = "This pull request changes no user-facing interface."
    rows = [f"| {SURFACES[s.name]} | {STATUS_TEXT[s.status]} |" for s in surfaces]
    ui_result = "Changed: the author picks the tier" if ui else "Not changed"
    rows.append(f"| Console and UI libraries | {ui_result} |")
    lines = [
        MARKER,
        f"## Review tier: `{tier}`",
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
        if s.status == "same":
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
        f"The classifier only raises the tier. It never removes `{HUMAN}`.",
        *([AUTO_BOT] if auto_bot else []),
    ]
    return "\n".join(lines) + "\n"


def gh(*args: str, stdin: str | None = None) -> str:
    return subprocess.run(
        ["gh", *args], input=stdin, capture_output=True, text=True, check=True
    ).stdout


def gh_lines(*args: str) -> list[Any]:
    return [json.loads(line) for line in gh(*args).splitlines() if line]


def bot_labelers(repo: str, *, number: int) -> list[str]:
    """Returns who labeled the pull request review/bot, oldest first."""
    return gh_lines(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/events",
        "--jq",
        f'.[] | select(.event == "labeled" and .label.name == "{BOT}")'
        " | .actor.login | tojson",
    )


def find_comment(repo: str, *, number: int) -> dict[str, Any] | None:
    comments = gh_lines(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/comments",
        "--jq",
        ".[] | {id, body}",
    )
    return next((c for c in comments if c["body"].startswith(MARKER)), None)


def write_comment(
    repo: str, *, number: int, comment: dict[str, Any] | None, body: str
) -> None:
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


def main() -> None:
    root = Path(sys.argv[1])
    repo = os.environ["GITHUB_REPOSITORY"]
    head_sha = os.environ["HEAD_SHA"]
    number = int((root / "pr" / "number").read_text().strip())
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
    comment = find_comment(repo, number=number)
    # A person who labels review/bot after the classifier makes the choice their own.
    was_auto_bot = (
        comment is not None
        and AUTO_BOT in comment["body"]
        and bot_labelers(repo, number=number)[-1:] == [APP]
    )
    ui = changes_ui(files)
    reasons = trust_reasons(pull, files=files)
    surfaces = read_surfaces(root)
    # A push during the reads leaves the reports stale: check the head before writing.
    pull = json.loads(gh("api", f"repos/{repo}/pulls/{number}"))
    if pull["head"]["sha"] != head_sha:
        print(f"#{number} moved past {head_sha}; a newer run classifies it")
        return
    current = {label["name"] for label in pull["labels"]}
    if ui and not has_chosen_tier(current, auto_bot=was_auto_bot):
        reasons.append("A UI file changed and the author chose no tier.")
    add, remove = plan_labels(current, human=needs_human(surfaces, reasons=reasons))
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
    kept = bool(current & HUMAN_TIERS)
    auto_bot = BOT in add or (was_auto_bot and BOT in current and BOT not in remove)
    write_comment(
        repo,
        number=number,
        comment=comment,
        body=render(surfaces, ui=ui, reasons=reasons, kept=kept, auto_bot=auto_bot),
    )
    print(f"#{number}: add {add}, remove {remove}")


if __name__ == "__main__":
    main()
