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
    python3 -m review compare <days>

Run it from `.github/scripts`. Environment: GH_TOKEN, GITHUB_REPOSITORY, PR, HEAD_SHA,
BASE_SHA, CHECK_RUN, RUN_URL, and BOT, the slug of the GitHub App the review posts as.
`run` also takes POLICY_DIR, a trusted checkout that holds `.sedosaur/`, and POLICY_SHA,
the commit its policy comes from; HEAD_DIR, a checkout of the head; REPLAY, `true` for a
review that posts nothing; and CLAUDE_CODE_OAUTH_TOKEN, the only variable that reaches
Claude Code. A replay needs no CHECK_RUN.

`run` writes a record of the review: when it ran, the pull request, the commits, the
policy commit, the usage, and the review. `post` posts the review in a record. `compare`
takes only GH_TOKEN and GITHUB_REPOSITORY, runs in a clone with full history, and prints
a Markdown comparison of the records of the last days with Greptile's reviews of the
same commits.
"""

import dataclasses
import json
import os
import subprocess
import sys
import tempfile
from datetime import UTC, datetime, timedelta
from pathlib import Path

from review import (
    PullRequest,
    Target,
    compare,
    gh,
    load,
    open_findings,
    parse,
    post,
    reviewed,
    run,
)


def claude(args: list[str], stdin: str, cwd: Path) -> str:
    with tempfile.TemporaryDirectory() as home:
        env = {
            "PATH": os.environ["PATH"],
            "HOME": home,
            "CLAUDE_CONFIG_DIR": home,
            "CLAUDE_CODE_OAUTH_TOKEN": os.environ["CLAUDE_CODE_OAUTH_TOKEN"],
        }
        return subprocess.run(
            args,
            input=stdin,
            cwd=cwd,
            env=env,
            stdout=subprocess.PIPE,
            text=True,
            check=True,
        ).stdout


def review(target: Target, path: Path, *, replay: bool) -> None:
    config = load(Path(os.environ["POLICY_DIR"]))
    head = Path(os.environ["HEAD_DIR"]).resolve()
    excluded = [f":(glob,exclude){g}" for g in config.excluded]
    diff = subprocess.run(
        [
            *("git", "-C", str(head), "diff", "--no-color", "--no-ext-diff"),
            *(f"{target.base}...{target.head}", "--", ".", *excluded),
        ],
        stdout=subprocess.PIPE,
        text=True,
        check=True,
    ).stdout
    since = next(
        (c for c in reviewed(target, gh) if _before(head, c, target.head)), None
    )
    new = ""
    if since is not None:
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
    meta = json.loads(
        gh(
            "api",
            f"repos/{target.repo}/pulls/{target.pr}",
            "--jq",
            '{title, body: (.body // "")}',
        )
    )
    pr = PullRequest(
        number=target.pr,
        title=meta["title"],
        body=meta["body"],
        head=head,
        diff=diff,
        # A replay of an old commit must not see the threads that came after it.
        open=set() if replay else open_findings(target, gh),
        new=new,
    )
    with tempfile.TemporaryDirectory() as workdir:
        result, usage = run(config, pr, claude=claude, workdir=Path(workdir))
    record = {
        "at": datetime.now(UTC).isoformat(timespec="seconds"),
        "pr": target.pr,
        "head": target.head,
        "base": target.base,
        "policy": os.environ["POLICY_SHA"],
        "usage": dataclasses.asdict(usage),
        "review": dataclasses.asdict(result),
    }
    path.write_text(json.dumps(record))


def _before(head: Path, commit: str, tip: str) -> bool:
    """Reports whether a commit comes before the tip in the history of a checkout. A
    commit that a force push dropped is not in the history."""
    if commit == tip:
        return False
    check = ("git", "-C", str(head), "merge-base", "--is-ancestor", commit, tip)
    return subprocess.run(check, capture_output=True).returncode == 0


def main(argv: list[str]) -> None:
    if len(argv) != 2 or argv[0] not in ("run", "post", "compare"):
        sys.exit(__doc__)
    env = os.environ
    if argv[0] == "compare":
        since = datetime.now(UTC) - timedelta(days=int(argv[1]))
        repo = env["GITHUB_REPOSITORY"]
        recs = compare.records(gh, repo, since)
        ps = compare.pairs(gh, compare.git, repo, recs)
        print(compare.report(ps, recs, since=since))
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
    path = Path(argv[1])
    if argv[0] == "run":
        review(target, path, replay=env.get("REPLAY") == "true")
    else:
        post(parse(json.loads(path.read_text())["review"]), target=target, gh=gh)


if __name__ == "__main__":
    main(sys.argv[1:])
