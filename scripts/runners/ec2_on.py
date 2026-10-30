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

"""Starts the EC2 instance behind a runner.

    uv run ec2_on.py windows-build-bot-i-0f44

dispatch.py calls run() when 1 is written to <runner>_ec2_cmd.
"""

import argparse

from aws import AWS


def run(runner: str, *, aws: AWS) -> str:
    iid, info = aws.instance(runner)
    aws.ec2[info["region"]].start_instances(InstanceIds=[iid])
    return f"{runner} ({info['region']}): start requested"


def main() -> None:
    p = argparse.ArgumentParser(description=(__doc__ or "").split("\n", 1)[0])
    p.add_argument("runner", help="runner name, e.g. windows-build-bot-i-0f44")
    print(run(p.parse_args().runner, aws=AWS()))


if __name__ == "__main__":
    main()
