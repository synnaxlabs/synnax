#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Posts a review to a pull request.

    python3 -m review post <review.json>

Run it from `.github/scripts`. Environment: GH_TOKEN, GITHUB_REPOSITORY, PR, HEAD_SHA,
BASE_SHA, CHECK_RUN, RUN_URL, and BOT, the slug of the GitHub App the review posts as.
"""

import json
import os
import sys
from pathlib import Path

from review import Target, gh, parse, post


def main(argv: list[str]) -> None:
    if len(argv) != 2 or argv[0] != "post":
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
    post(parse(json.loads(Path(argv[1]).read_text())), target=target, gh=gh)


if __name__ == "__main__":
    main(sys.argv[1:])
