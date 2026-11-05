#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Renders a review as the Markdown that GitHub shows on a pull request."""

import re
from collections import Counter

from review.findings import SEVERITIES, Finding, Review

BADGES = "https://raw.githubusercontent.com/synnaxlabs/synnax/main/.sedosaur/badges"
SUMMARY_START = "<!-- sedosaur_comment -->"
SUMMARY_END = "<!-- /sedosaur_comment -->"
OUTSIDE = "<!-- sedosaur_outside_diff -->"
VOTE = "<sub>React 👍 if this is a real problem or 👎 if it is noise.</sub>"
SUBJECT_CHARS = 48


def badge(name: str, alt: str) -> str:
    """Renders the light and dark variants of a badge as one image."""
    scheme = 'media="(prefers-color-scheme: dark)"'
    dark = f'<source {scheme} srcset="{BADGES}/{name}-dark.svg">'
    light = f'<img alt="{alt}" src="{BADGES}/{name}.svg" align="top">'
    return f"<picture>{dark}{light}</picture>"


def fence(text: str, info: str = "") -> str:
    """Fences text in more backticks than any run inside it."""
    longest = max((len(run) for run in re.findall(r"`+", text)), default=0)
    ticks = "`" * max(3, longest + 1)
    return f"{ticks}{info}\n{text}\n{ticks}"


def heading(f: Finding) -> str:
    return f"{badge(f.severity.lower(), f.severity)} **{f.title}**"


def title(body: str) -> str | None:
    """Returns the title of a finding from the comment that heading started."""
    m = re.search(r"\*\*(.+?)\*\*", body.split("\n", 1)[0])
    return m[1] if m else None


def comment(f: Finding, *, blob: str) -> str:
    """Renders a finding as an inline review comment.

    :param blob: The URL that a repository path appends to, at the reviewed commit.
    """
    parts = [heading(f), f.body]
    if f.suggestion is not None:
        parts.append(fence(f.suggestion, "suggestion"))
    if f.rule is not None:
        parts.append(f"**Rule:** [`{f.rule}`]({blob}/{f.rule})")
    parts.append(VOTE)
    return "\n\n".join(parts)


def outside(findings: list[Finding], *, blob: str) -> str:
    """Renders the comment that lists findings on lines outside the diff."""
    lines = [OUTSIDE, "", "<h3>Comments Outside Diff</h3>", ""]
    if not findings:
        return "\n".join([*lines, "No findings outside the diff remain."]) + "\n"
    lines += ["These findings could not be posted inline.", ""]
    for f in findings:
        where = f"[`{f.path}:{f.line}`]({blob}/{f.path}#L{f.line})"
        body = f.body.replace("\n", "\n  ")
        lines += [f"- {heading(f)} {where}", "", f"  {body}", ""]
    return "\n".join(lines)


def subject(message: str) -> str:
    """Shortens a commit message to a link label that Markdown keeps intact."""
    line = message.split("\n", 1)[0]
    if len(line) > SUBJECT_CHARS:
        line = line[: SUBJECT_CHARS - 3].rsplit(" ", 1)[0] + "..."
    return re.sub(r"([\[\]])", r"\\\1", line)


def reviews(body: str) -> int:
    """Returns how many reviews the summary in a pull request body counts."""
    m = re.search(rf"{re.escape(SUMMARY_START)}.*?Reviews \((\d+)\)", body, re.S)
    return int(m[1]) if m else 0


def summary(
    review: Review, *, count: int, commit: str, message: str, run_url: str
) -> str:
    """Renders the summary that heads the pull request body.

    :param count: How many reviews the pull request has had, this one included.
    :param commit: The URL of the reviewed commit.
    :param message: The message of the reviewed commit.
    """
    n = review.score
    bullets = [f"- {line}" for line in review.summary]
    footer = (
        f"<sub>Reviews ({count}) · Last reviewed commit: "
        f"[{subject(message)}]({commit}) · [Run]({run_url})</sub>"
    )
    return "\n".join(
        [
            SUMMARY_START,
            "",
            f"<h2>{badge(f'score-{n}', f'{n} of 5')} Confidence Score: {n}/5</h2>",
            "",
            review.verdict,
            "",
            "<h3>Summary</h3>",
            "",
            *bullets,
            "",
            f"<!-- sedosaur_confidence_score:{n} -->",
            "",
            footer,
            "",
            SUMMARY_END,
        ]
    )


def splice(body: str, block: str) -> str:
    """Replaces the summary in a pull request body, or appends it if there is none."""
    start, end = body.find(SUMMARY_START), body.find(SUMMARY_END)
    if start == -1 or end == -1:
        return f"{body.rstrip()}\n\n{block}\n" if body.strip() else f"{block}\n"
    return body[:start] + block + body[end + len(SUMMARY_END) :]


def check(review: Review) -> dict[str, str]:
    """Renders the output of the review's check run."""
    counts = Counter(f.severity for f in review.findings)
    found = ", ".join(f"{counts[s]} {s}" for s in SEVERITIES if counts[s])
    return {
        "title": f"Confidence Score: {review.score}/5",
        "summary": f"{review.verdict}\n\nFindings: {found or 'none'}.",
    }
