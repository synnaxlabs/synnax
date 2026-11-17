#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Stops the EC2 instance behind a runner that GitHub shows offline with no job.

    uv run ec2_off.py windows-build-bot-i-0f44

dispatch.py calls run() when 0 is written to <runner>_ec2_cmd.
"""

import argparse
from collections.abc import Mapping
from typing import Protocol

from aws import AWS
from common import parameter_store, secret
from github import GitHub


class Stopper(Protocol):
    def stop_instances(self, *, InstanceIds: list[str]) -> object: ...


class Fleet(Protocol):
    """The AWS calls run() makes, as aws.AWS has them."""

    @property
    def ec2(self) -> Mapping[str, Stopper]: ...

    def instance(self, runner: str) -> tuple[str, dict]: ...


def run(runner: str, *, aws: Fleet, github: GitHub) -> str | None:
    # An online runner can take a job before the instance goes down, so its runner is
    # stopped first, with <runner>_gh_cmd.
    if not github.offline(runner):
        return f"{runner}: online or busy, stop skipped"
    iid, info = aws.instance(runner)
    aws.ec2[info["region"]].stop_instances(InstanceIds=[iid])
    return None


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    p.add_argument("runner", help="runner name, e.g. windows-build-bot-i-0f44")
    store = parameter_store()
    github = GitHub(secret("GH_APP_ID", store=store), secret("GH_APP_KEY", store=store))
    if (skipped := run(p.parse_args().runner, aws=AWS(), github=github)) is not None:
        print(skipped)


if __name__ == "__main__":
    main()
