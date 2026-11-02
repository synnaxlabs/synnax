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
# dependencies = ["boto3", "certifi>=2026.10.19", "synnax>=0.59,<0.60"]
# ///

"""Stops the EC2 instance behind a runner, unless GitHub has a job assigned to it.

    uv run ec2_off.py windows-build-bot-i-0f44

dispatch.py calls run() when 0 is written to <runner>_ec2_cmd.
"""

import argparse

from aws import AWS
from common import parameter_store, secret
from github import GitHub


def run(runner: str, *, aws: AWS, github: GitHub) -> str:
    # The scaler decides on a status that trails GitHub by seconds, in which time an
    # idle runner can take a job.
    if github.busy(runner):
        return f"{runner}: busy, stop skipped"
    iid, info = aws.instance(runner)
    aws.ec2[info["region"]].stop_instances(InstanceIds=[iid])
    return f"{runner} ({info['region']}): stop requested"


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    p.add_argument("runner", help="runner name, e.g. windows-build-bot-i-0f44")
    github = GitHub(secret("GH_TOKEN", store=parameter_store()))
    print(run(p.parse_args().runner, aws=AWS(), github=github))


if __name__ == "__main__":
    main()
