#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Files a QA report as one GitHub issue, with its screenshots shown inline.

Usage: publish_qa_report.py DIR [--label LABEL] [--dry-run]

DIR holds `report.md`, whose first line is `# <issue title>`, and the PNG screenshots
it links as `images/<name>.png`. The script pushes the linked screenshots to the
`agent-assets` branch, points the links at them, and opens the issue with `gh`. It
prints the issue URL. With --dry-run it writes the issue to `DIR/issue.md` instead,
and touches neither Git nor GitHub.
"""

import argparse
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

ASSETS_BRANCH = "agent-assets"
IMAGE_LINK = re.compile(r"\]\((images/[\w.-]+\.png)\)")


def git(*args: str, env: dict[str, str] | None = None) -> str:
    return subprocess.run(
        ["git", *args], check=True, stdout=subprocess.PIPE, text=True, env=env
    ).stdout.strip()


def upload(paths: list[Path], *, prefix: str, repo: str) -> str:
    """Commits paths under prefix on the assets branch and returns their base URL.

    Retries when another publisher moves the branch between the fetch and the push.
    """
    ref = f"refs/heads/{ASSETS_BRANCH}"
    for _ in range(3):
        env = {**os.environ, "GIT_INDEX_FILE": str(Path(tempfile.mkdtemp()) / "i")}
        parent: list[str] = []
        ls = ["git", "ls-remote", "--exit-code", "origin", ref]
        if subprocess.run(ls, stdout=subprocess.DEVNULL).returncode == 0:
            git("fetch", "--quiet", "origin", ref)
            git("read-tree", "FETCH_HEAD", env=env)
            parent = ["-p", "FETCH_HEAD"]
        for path in paths:
            blob = git("hash-object", "-w", str(path))
            info = f"100644,{blob},{prefix}/{path.name}"
            git("update-index", "--add", "--cacheinfo", info, env=env)
        tree = git("write-tree", env=env)
        commit = git("commit-tree", tree, *parent, "-m", f"Add images for {prefix}")
        push = ["git", "push", "--quiet", "origin", f"{commit}:{ref}"]
        if subprocess.run(push).returncode == 0:
            return f"https://raw.githubusercontent.com/{repo}/{commit}/{prefix}"
    sys.exit(f"failed to push images to {ASSETS_BRANCH} three times")


def main() -> int:
    p = argparse.ArgumentParser()
    p.add_argument("dir", type=Path)
    p.add_argument("--label", default="console-qa")
    p.add_argument("--dry-run", action="store_true")
    a = p.parse_args()

    title, _, body = (a.dir / "report.md").read_text().partition("\n")
    if not title.startswith("# "):
        sys.exit("report.md must start with a `# <title>` line")
    title = title[2:].strip()
    links = sorted(set(IMAGE_LINK.findall(body)))
    missing = [link for link in links if not (a.dir / link).is_file()]
    if missing:
        sys.exit(f"report.md links missing images: {', '.join(missing)}")
    repo = os.environ.get("GITHUB_REPOSITORY", "synnaxlabs/synnax")
    run = os.environ.get("GITHUB_RUN_ID")
    if run:
        server = os.environ.get("GITHUB_SERVER_URL", "https://github.com")
        body += f"\n\n[Workflow run]({server}/{repo}/actions/runs/{run})\n"

    if a.dry_run:
        out = a.dir / "issue.md"
        out.write_text(f"# {title}\n{body}")
        print(out)
        return 0

    if links:
        prefix = f"console-qa-{run or os.getpid()}"
        base = upload([a.dir / link for link in links], prefix=prefix, repo=repo)
        body = IMAGE_LINK.sub(lambda m: f"]({base}/{Path(m[1]).name})", body)
    body_file = a.dir / "issue.md"
    body_file.write_text(body)
    cmd = ["gh", "issue", "create", "--title", title, "--label", a.label]
    cmd += ["--body-file", str(body_file)]
    done = subprocess.run(cmd, check=True, stdout=subprocess.PIPE, text=True)
    print(done.stdout.strip())
    return 0


if __name__ == "__main__":
    sys.exit(main())
