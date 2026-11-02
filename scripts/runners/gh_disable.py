#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

# /// script
# requires-python = ">=3.12,<4"
# dependencies = ["boto3", "synnax>=0.59,<0.60"]
# ///

"""Stops the runner service on an instance over SSM. Cancels a job in progress.

    uv run gh_disable.py windows-build-bot-i-0f44

By hand only; agent.py on the runner acts on <runner>_gh_cmd.
"""

import argparse

from aws import AWS

LINUX = ["systemctl stop github-runner.service"]
# Stopping the task alone leaves Runner.Listener up.
WINDOWS = [
    "Stop-ScheduledTask -TaskName 'GitHubActionsRunner' -ErrorAction SilentlyContinue",
    "Stop-Process -Name 'Runner.Listener' -Force -ErrorAction SilentlyContinue",
]


def run(runner: str, *, aws: AWS) -> str:
    iid, info = aws.instance(runner)
    if not info["running"]:
        return f"{iid} is {info['state']}, nothing to disable"
    result = aws.run(iid, info=info, linux=LINUX, windows=WINDOWS)
    return f"runner stop sent to {iid} ({result})"


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    p.add_argument("runner", help="runner name, e.g. windows-build-bot-i-0f44")
    print(run(p.parse_args().runner, aws=AWS()))


if __name__ == "__main__":
    main()
