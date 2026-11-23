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
it links as `images/<name>.png`. The script stores the linked screenshots in Vercel
Blob with the token in BLOB_READ_WRITE_TOKEN, points the links at them, and opens the
issue with `gh`. It prints the issue URL. With --dry-run it writes the issue to
`DIR/issue.md` instead, and touches neither Vercel Blob nor GitHub. Run it with
`scripts` on PYTHONPATH.
"""

import argparse
import os
import re
import subprocess
import sys
from pathlib import Path

from agent.agent import BLOB_API, upload

IMAGE_LINK = re.compile(r"\]\((images/[\w.-]+\.png)\)")


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

    token = os.environ.get("BLOB_READ_WRITE_TOKEN")
    if not token:
        sys.exit("BLOB_READ_WRITE_TOKEN must hold the Vercel Blob read-write token")
    api = os.environ.get("VERCEL_BLOB_API_URL", BLOB_API)
    paths = [a.dir / link for link in links]
    urls = upload(paths, prefix="console-qa", token=token, api=api)
    hosted = dict(zip(links, urls, strict=True))
    body = IMAGE_LINK.sub(lambda m: f"]({hosted[m[1]]})", body)
    body_file = a.dir / "issue.md"
    body_file.write_text(body)
    cmd = ["gh", "issue", "create", "--title", title, "--label", a.label]
    cmd += ["--body-file", str(body_file)]
    done = subprocess.run(cmd, check=True, stdout=subprocess.PIPE, text=True)
    print(done.stdout.strip())
    return 0


if __name__ == "__main__":
    sys.exit(main())
