#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Reviews a pull request, or posts a review to it.

    python3 -m review run <review.json>
    python3 -m review post <review.json>
    python3 -m review carry
    python3 -m review dismiss
    python3 -m review judge <judgments.json>
    python3 -m review rule <judgments.json>
    python3 -m review compare <days> <records>

Run it from `.github/scripts`. Environment: GH_TOKEN, GITHUB_REPOSITORY, PR, HEAD_SHA,
BASE_SHA, CHECK_RUN, RUN_URL, and BOT, the slug of the GitHub App the review posts as.
`run` also takes POLICY_DIR, a trusted checkout that holds `.sedosaur/`, and POLICY_SHA,
the commit its policy comes from; HEAD_DIR, a checkout of the head; REPLAY, `true` for a
review that posts nothing; and CLAUDE_CODE_OAUTH_TOKEN, the only secret that reaches
Claude Code. A replay needs no CHECK_RUN.

`run` writes a record of the review: when it ran, the pull request, the commits, the
policy commit, the usage, the review, and each pass before the join. `post` posts the
review in a record. `carry` gives the head the newest review of the pull request when
the diff did not change, completes CHECK_RUN with it, and prints `true` if it did. It
also takes KEY, the external ID of the check run, `<pr>:<base ref>`. `dismiss` answers
COMMENT, a reply on the thread of one of the bot's findings, or a comment that dismisses
a finding outside the diff: an `@sedosaur dismiss` from someone with write access
dismisses the finding, and scores the head 5/5 when no finding of its review is left.
It also takes KEY, and prints `true` when a rebuttal on a thread waits for `judge`.
`judge` takes POLICY_DIR, HEAD_DIR, and CLAUDE_CODE_OAUTH_TOKEN as `run` does, judges
each rebuttal with Claude Code, and writes the judgments. It fails after it writes them
when one rebuttal could not be judged. `rule` posts them, and takes KEY. `compare`
takes only GH_TOKEN and GITHUB_REPOSITORY, runs in a clone with full history, and
prints a Markdown comparison of the records of the last days with Greptile's reviews
of the same commits. It reads each record from the check run that `post` completed,
keeps it in the records directory, and reads it from there on the next run.

When Claude Code stops at the usage limit of its login, `run` and `judge` write the
reset time, in Unix seconds, as the `limited-until` step output. When Claude rejects the
token of the login, they write the time Claude Code started as `login-broken`. Both then
fail.
"""

import contextlib
import dataclasses
import json
import os
import re
import subprocess
import sys
import tempfile
from collections.abc import Iterator, Sequence
from datetime import UTC, datetime, timedelta
from pathlib import Path

from review import (
    Judgment,
    LoginRejected,
    PullRequest,
    Target,
    UsageLimit,
    authority,
    carry,
    compare,
    dismiss,
    dismissals,
    gh,
    judge,
    load,
    outside_findings,
    post,
    reviewed,
    rule,
    run,
    threads,
)

_FILE = re.compile(r"diff --git a/.* b/(.*)")
_INDEX = re.compile(r"(?m)^index .*\n")
_BINARY = re.compile(r"(?m)^Binary files ")
_HUNK = re.compile(r"(?m)^@@ -\d+(?:,\d+)? \+\d+(?:,\d+)? @@")


def claude(args: list[str], stdin: str, cwd: Path) -> Iterator[str]:
    with tempfile.TemporaryDirectory() as home:
        env = {
            "PATH": os.environ["PATH"],
            "HOME": home,
            "CLAUDE_CONFIG_DIR": home,
            "CLAUDE_CODE_OAUTH_TOKEN": os.environ["CLAUDE_CODE_OAUTH_TOKEN"],
            # A subscription writes the one-hour cache, at 1.6 times the price of the
            # five-minute one. The turns of a review come seconds apart, and no other
            # review shares its cache, so the hour is never used.
            "CLAUDE_CODE_PROMPT_CACHE_TTL": "5m",
        }
        with subprocess.Popen(
            args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            cwd=cwd,
            env=env,
            text=True,
        ) as proc:
            assert proc.stdin is not None
            assert proc.stdout is not None
            # The exit code tells why Claude Code stopped reading its stdin.
            with contextlib.suppress(BrokenPipeError):
                proc.stdin.write(stdin)
            with contextlib.suppress(BrokenPipeError):
                proc.stdin.close()
            yield from proc.stdout
        if proc.returncode:
            raise subprocess.CalledProcessError(proc.returncode, args)


def review(target: Target, path: Path, *, replay: bool) -> None:
    config = load(Path(os.environ["POLICY_DIR"]))
    head = Path(os.environ["HEAD_DIR"]).resolve()
    excluded = _excluded(config.excluded)
    diff = _diff(head, target.base, target.head, excluded)
    earlier = [c for c in reviewed(target, gh) if c != target.head]
    since = next((c for c in earlier if _before(head, c, target.head)), None)
    # The last review is the newest one, even when a force push dropped its commit. A
    # replay cannot tell such a commit from a review that came after it.
    if not replay and earlier and since != earlier[0]:
        since = None
    rewritten = since is None and bool(earlier) and not replay
    new = ""
    dropped: list[str] = []
    # A merge can change the diff without a commit of its own in the log.
    if since is not None and not _merges(head, since, target.head):
        new = subprocess.run(
            [
                *("git", "-C", str(head), "log", "--first-parent", "--no-merges"),
                *(
                    "--patch",
                    "--no-color",
                    "--no-ext-diff",
                    "--format=commit %H%n%n%s%n",
                ),
                *(f"{since}..{target.head}", "--", ".", *excluded),
            ],
            stdout=subprocess.PIPE,
            text=True,
            check=True,
        ).stdout
    elif since is not None or rewritten:
        old = since or earlier[0]
        new = diff
        if _fetched(head, old):
            new, dropped = _changes(diff, _diff(head, target.base, old, excluded))
    meta = json.loads(
        gh(
            "api",
            f"repos/{target.repo}/pulls/{target.pr}",
            "--jq",
            '{title, body: (.body // "")}',
        )
    )
    # A replay of an old commit must not see the threads that came after it.
    found = [] if replay else threads(target, gh)
    held = set() if replay else outside_findings(target, gh)
    pr = PullRequest(
        number=target.pr,
        title=meta["title"],
        body=meta["body"],
        head=head,
        diff=diff,
        open={(th.path, th.title) for th in found if not th.resolved} | held,
        dismissed=[]
        if replay
        else dismissals(target, gh, found, authority(target, gh)),
        new=new,
        dropped=dropped,
        reviewed=since is not None or rewritten,
    )
    with tempfile.TemporaryDirectory() as workdir:
        result, usage, passes = run(config, pr, claude=claude, workdir=Path(workdir))
    record = {
        "at": datetime.now(UTC).isoformat(timespec="seconds"),
        "pr": target.pr,
        "head": target.head,
        "base": target.base,
        "policy": os.environ["POLICY_SHA"],
        "usage": dataclasses.asdict(usage),
        "review": dataclasses.asdict(result),
        "passes": [dataclasses.asdict(p) for p in passes],
    }
    path.write_text(json.dumps(record))


def judgments(target: Target, path: Path) -> None:
    config = load(Path(os.environ["POLICY_DIR"]))
    head = Path(os.environ["HEAD_DIR"]).resolve()
    diff = _diff(head, target.base, target.head, _excluded(config.excluded))
    with tempfile.TemporaryDirectory() as workdir:
        found, failed = judge(
            config,
            target,
            gh,
            head=head,
            diff=diff,
            claude=claude,
            workdir=Path(workdir),
        )
    path.write_text(json.dumps([dataclasses.asdict(j) for j in found]))
    if failed:
        raise ExceptionGroup("some rebuttals could not be judged", failed)


def _excluded(globs: list[str]) -> list[str]:
    return [f":(glob,exclude){g}" for g in globs]


def _diff(head: Path, base: str, commit: str, excluded: list[str]) -> str:
    """Returns the diff from the merge base of the base and a commit to the commit."""
    return subprocess.run(
        [
            *("git", "-C", str(head), "diff", "--no-color", "--no-ext-diff"),
            *(f"{base}...{commit}", "--", ".", *excluded),
        ],
        stdout=subprocess.PIPE,
        text=True,
        check=True,
    ).stdout


def _fetched(head: Path, commit: str) -> bool:
    """Fetches a commit into a checkout when it is not there, as after a force push.
    Returns False when the remote no longer has it."""
    git = ("git", "-C", str(head))
    there = [*git, "cat-file", "-e", f"{commit}^{{commit}}"]
    if subprocess.run(there, capture_output=True).returncode == 0:
        return True
    fetch = [*git, "fetch", "--no-tags", "-q", "origin", commit]
    return subprocess.run(fetch, capture_output=True).returncode == 0


def _files(diff: str) -> dict[str, tuple[str, str]]:
    """Splits a diff by file. Maps each path to its patch, and to the patch without its
    line numbers and, unless only they show a binary file's change, its blob IDs."""
    out: dict[str, tuple[str, str]] = {}
    for chunk in re.split(r"(?m)^(?=diff --git )", diff):
        if m := _FILE.match(chunk):
            key = chunk if _BINARY.search(chunk) else _INDEX.sub("", chunk)
            out[m[1]] = chunk, _HUNK.sub("@@", key)
    return out


def _changes(diff: str, before: str) -> tuple[str, list[str]]:
    """Returns the files of a diff whose patches differ from those of an earlier diff,
    and the files that only the earlier diff changes. Blob IDs and line numbers do not
    count."""
    was, now = _files(before), _files(diff)
    kept = (c for p, (c, key) in now.items() if p not in was or was[p][1] != key)
    return "".join(kept), sorted(was.keys() - now.keys())


def _merges(head: Path, since: str, tip: str) -> bool:
    """Reports whether a merge commit lies between two commits of a checkout."""
    log = ("git", "-C", str(head), "rev-list", "--first-parent", "--merges")
    out = subprocess.run(
        [*log, f"{since}..{tip}"], capture_output=True, text=True, check=True
    )
    return bool(out.stdout.strip())


def _before(head: Path, commit: str, tip: str) -> bool:
    """Reports whether a commit comes before the tip in the history of a checkout. A
    commit that a force push dropped is not in the history."""
    check = ("git", "-C", str(head), "merge-base", "--is-ancestor", commit, tip)
    return subprocess.run(check, capture_output=True).returncode == 0


def _leaves[E: BaseException](g: BaseExceptionGroup[E]) -> Sequence[E]:
    return [
        leaf
        for e in g.exceptions
        for leaf in (_leaves(e) if isinstance(e, BaseExceptionGroup) else [e])
    ]


def _output(name: str, value: int) -> None:
    with Path(os.environ["GITHUB_OUTPUT"]).open("a") as f:
        f.write(f"{name}={value}\n")


def _claude(target: Target, command: str, path: Path) -> None:
    try:
        if command == "judge":
            judgments(target, path)
        else:
            review(target, path, replay=os.environ.get("REPLAY") == "true")
    except* UsageLimit as g:
        _output("limited-until", max(e.until for e in _leaves(g)))
        raise
    except* LoginRejected as g:
        _output("login-broken", min(e.started for e in _leaves(g)))
        raise


def main(argv: list[str]) -> None:
    args = {
        "run": 2,
        "post": 2,
        "carry": 1,
        "dismiss": 1,
        "judge": 2,
        "rule": 2,
        "compare": 3,
    }
    if not argv or args.get(argv[0]) != len(argv):
        sys.exit(__doc__)
    env = os.environ
    if argv[0] == "compare":
        since = datetime.now(UTC) - timedelta(days=int(argv[1]))
        repo = env["GITHUB_REPOSITORY"]
        cache = Path(argv[2])
        ps = compare.pairs(gh, compare.git, repo, since, cache)
        print(compare.report(ps, compare.limits(gh, repo, since, cache), since=since))
        return
    target = Target(
        repo=env["GITHUB_REPOSITORY"],
        pr=int(env["PR"]),
        head=env["HEAD_SHA"],
        base=env["BASE_SHA"],
        check_run=int(env["CHECK_RUN"]) if env.get("CHECK_RUN") else None,
        run_url=env["RUN_URL"],
        bot=env["BOT"],
    )
    if argv[0] == "carry":
        print(str(carry(target, gh, key=env["KEY"])).lower())
        return
    if argv[0] == "dismiss":
        waits = dismiss(target, gh, comment=int(env["COMMENT"]), key=env["KEY"])
        print(str(waits).lower())
        return
    path = Path(argv[1])
    if argv[0] in ("judge", "run"):
        _claude(target, argv[0], path)
    elif argv[0] == "rule":
        found = [Judgment(**j) for j in json.loads(path.read_text())]
        rule(target, gh, found, key=env["KEY"])
    else:
        post(json.loads(path.read_text()), target=target, gh=gh)


if __name__ == "__main__":
    main(sys.argv[1:])
