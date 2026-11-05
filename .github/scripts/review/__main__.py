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

Run it from `.github/scripts`. Environment: GH_TOKEN, GITHUB_REPOSITORY, PR, HEAD_SHA,
BASE_SHA, CHECK_RUN, RUN_URL, and BOT, the slug of the GitHub App the review posts as.
`run` also takes BASE_DIR, a checkout of the base branch; HEAD_DIR, a checkout of the
head; and CLAUDE_CODE_OAUTH_TOKEN, the only variable that reaches Claude Code.
"""

import dataclasses
import json
import os
import subprocess
import sys
import tempfile
from pathlib import Path

from review import PullRequest, Target, gh, load, open_findings, parse, post, run


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


def review(target: Target, path: Path) -> None:
    config = load(Path(os.environ["BASE_DIR"]))
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
        open=open_findings(target, gh),
    )
    with tempfile.TemporaryDirectory() as workdir:
        result = run(config, pr, claude=claude, workdir=Path(workdir))
    path.write_text(json.dumps(dataclasses.asdict(result)))


def main(argv: list[str]) -> None:
    if len(argv) != 2 or argv[0] not in ("run", "post"):
        sys.exit(__doc__)
    env = os.environ
    target = Target(
        repo=env["GITHUB_REPOSITORY"],
        pr=int(env["PR"]),
        head=env["HEAD_SHA"],
        base=env["BASE_SHA"],
        check_run=int(env["CHECK_RUN"]),
        run_url=env["RUN_URL"],
        bot=env["BOT"],
    )
    path = Path(argv[1])
    if argv[0] == "run":
        review(target, path)
    else:
        post(parse(json.loads(path.read_text())), target=target, gh=gh)


if __name__ == "__main__":
    main(sys.argv[1:])
