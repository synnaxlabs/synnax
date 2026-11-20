#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Brings the agent to the deployed commit, then runs agent.py with the same arguments.

Runs on each runner, started at boot by the runner-agent systemd unit (Ubuntu) or the
RunnerAgent scheduled task (Windows). The Deploy - Runners workflow stores the deployed
commit in the Parameter Store parameter /runner-agents/commit, so a runner that was
stopped during a deploy updates when it starts. When the update fails, the installed
agent runs.

    uv run --project . --frozen --no-dev agent_boot.py --host <address> --secure
"""

import os
import subprocess
import sys
from collections.abc import Callable, Mapping
from pathlib import Path
from typing import Any, Protocol
from urllib.request import urlopen

import boto3
from botocore.config import Config

# pyproject.toml and uv.lock pin the dependencies that agent.py runs with.
FILES = ("agent.py", "agent_log.py", "gh_runner.py", "pyproject.toml", "uv.lock")
COMMIT_PARAMETER = "/runner-agents/commit"
SOURCE = "https://raw.githubusercontent.com/synnaxlabs/synnax/{}/scripts/runners/{}"
# The file that holds the commit of the installed agent.
INSTALLED = "agent_commit.txt"
# A machine that cannot reach AWS runs the installed agent in under a minute.
TIMEOUTS = Config(connect_timeout=10, read_timeout=10, retries={"max_attempts": 2})
DOWNLOAD_TIMEOUT = 30


class ParameterStore(Protocol):
    def get_parameter(self, *, Name: str) -> Mapping[str, Any]: ...


def download(url: str) -> bytes:
    with urlopen(url, timeout=DOWNLOAD_TIMEOUT) as response:
        return bytes(response.read())


def update(
    directory: Path, *, store: ParameterStore, fetch: Callable[[str], bytes]
) -> str | None:
    """Downloads the agent into directory when the deployed commit is not the installed
    one.

    :returns: the commit it installed, or None when the agent was up to date.
    """
    commit = str(store.get_parameter(Name=COMMIT_PARAMETER)["Parameter"]["Value"])
    installed = directory / INSTALLED
    if installed.exists() and installed.read_text().strip() == commit:
        return None
    # Every download ends before the first write, so a failure keeps the old agent.
    contents = {name: fetch(SOURCE.format(commit, name)) for name in FILES}
    # Atomic renames, run back to back, keep pyproject.toml and uv.lock from drifting
    # apart when an update stops partway.
    for name, content in contents.items():
        (directory / f"{name}.tmp").write_bytes(content)
    for name in contents:
        os.replace(directory / f"{name}.tmp", directory / name)
    installed.write_text(commit)
    return commit


def main() -> int:
    directory = Path(__file__).parent
    try:
        store = boto3.client("ssm", region_name="us-east-1", config=TIMEOUTS)
        if commit := update(directory, store=store, fetch=download):
            print(f"installed the agent at {commit}", flush=True)
    except Exception as e:
        print(f"update failed: {type(e).__name__}: {e}", flush=True)
    uv = directory / ("uv.exe" if sys.platform == "win32" else "uv")
    agent = directory / "agent.py"
    command = [str(uv), "run", "--project", str(directory), "--locked", "--no-dev"]
    command.append(str(agent))
    command.extend(sys.argv[1:])
    return subprocess.run(command, check=False).returncode


if __name__ == "__main__":
    sys.exit(main())
