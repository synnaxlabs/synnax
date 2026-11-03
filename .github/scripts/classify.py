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
head. No report covers the Console, so the author picks the tier of a UI change, and
one with no tier gets review/human. The script only raises: it never removes
review/human. It reads the pull request number from <artifacts-dir>/pr/number and
refuses it unless the pull request's head is HEAD_SHA. Environment: GH_TOKEN,
GITHUB_REPOSITORY, and HEAD_SHA.
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
) -> str:
    changed = any(s.status != "same" for s in surfaces)
    tier = HUMAN if changed or reasons or kept else BOT
    if changed:
        summary = "This pull request changes a user-facing interface."
    elif reasons:
        summary = "The classifier cannot clear this pull request."
    elif kept:
        summary = "No interface changed, but a human review tier is already set."
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
    ]
    return "\n".join(lines) + "\n"


def gh(*args: str, stdin: str | None = None) -> str:
    return subprocess.run(
        ["gh", *args], input=stdin, capture_output=True, text=True, check=True
    ).stdout


def gh_lines(*args: str) -> list[Any]:
    return [json.loads(line) for line in gh(*args).splitlines() if line]


def upsert_comment(repo: str, *, number: int, body: str) -> None:
    comments = gh_lines(
        "api",
        "--paginate",
        f"repos/{repo}/issues/{number}/comments",
        "--jq",
        ".[] | {id, body}",
    )
    payload = json.dumps({"body": body})
    for c in comments:
        if c["body"].startswith(MARKER):
            gh(
                "api",
                "-X",
                "PATCH",
                f"repos/{repo}/issues/comments/{c['id']}",
                "--input",
                "-",
                stdin=payload,
            )
            return
    gh(
        "api",
        "-X",
        "POST",
        f"repos/{repo}/issues/{number}/comments",
        "--input",
        "-",
        stdin=payload,
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
    current = {label["name"] for label in pull["labels"]}
    ui = changes_ui(files)
    reasons = trust_reasons(pull, files=files)
    if ui and not current & (HUMAN_TIERS | {BOT}):
        reasons.append("A UI file changed and the author set no tier.")
    surfaces = read_surfaces(root)
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
    upsert_comment(
        repo,
        number=number,
        body=render(surfaces, ui=ui, reasons=reasons, kept=kept),
    )
    print(f"#{number}: add {add}, remove {remove}")


if __name__ == "__main__":
    main()
