#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""The EC2 instances behind the runners, SSM to run commands on them, and the billing
APIs cost.py reads."""

from concurrent.futures import ThreadPoolExecutor
from typing import Any

import boto3
import synnax as sy
from botocore.config import Config

from common import instance_for

# Region to the abbreviation used in channel names.
REGIONS = {"us-east-1": "use1", "us-east-2": "use2"}
TIMEOUTS = Config(connect_timeout=10, read_timeout=30, retries={"max_attempts": 2})


class AWS:
    """Per-region EC2, SSM, and CloudWatch clients plus the global Cost Explorer and
    Pricing clients, safe to share across threads."""

    def __init__(self) -> None:
        self.ec2 = {
            r: boto3.client("ec2", region_name=r, config=TIMEOUTS) for r in REGIONS
        }
        self.ssm = {
            r: boto3.client("ssm", region_name=r, config=TIMEOUTS) for r in REGIONS
        }
        self.cloudwatch = {
            r: boto3.client("cloudwatch", region_name=r, config=TIMEOUTS)
            for r in REGIONS
        }
        self.ce = boto3.client("ce", region_name="us-east-1", config=TIMEOUTS)
        self.pricing = boto3.client("pricing", region_name="us-east-1", config=TIMEOUTS)

    def instances(self) -> dict[str, dict]:
        """Maps instance id to its region, state, running flag, whether it runs
        Windows, its type, and when it was launched."""
        found: dict[str, dict] = {}
        with ThreadPoolExecutor(len(REGIONS)) as pool:
            for region, pages in zip(
                REGIONS,
                pool.map(self._describe_instances, REGIONS),
                strict=True,
            ):
                for page in pages:
                    for res in page["Reservations"]:
                        for i in res["Instances"]:
                            state = i["State"]["Name"]
                            found[i["InstanceId"]] = {
                                "region": region,
                                "state": state,
                                "running": state == "running",
                                "windows": i.get("PlatformDetails") == "Windows",
                                "type": i["InstanceType"],
                                "launched": sy.TimeStamp(i["LaunchTime"]),
                            }
        return found

    def volumes(self) -> dict[str, int]:
        """GiB of volumes attached to each instance id."""
        sizes: dict[str, int] = {}
        with ThreadPoolExecutor(len(REGIONS)) as pool:
            for pages in pool.map(self._describe_volumes, REGIONS):
                for page in pages:
                    for v in page["Volumes"]:
                        for a in v["Attachments"]:
                            iid = a["InstanceId"]
                            sizes[iid] = sizes.get(iid, 0) + v["Size"]
        return sizes

    def instance(self, runner: str) -> tuple[str, dict]:
        """The instance behind a runner: its id and its instances() entry."""
        found = self.instances()
        iid = instance_for(runner, found=found)
        if iid is None:
            raise LookupError(f"no EC2 instance matches {runner}")
        return iid, found[iid]

    def run(self, iid: str, *, info: dict, linux: list[str], windows: list[str]) -> str:
        """Sends the script for the instance's OS over SSM; returns the command id."""
        if info["windows"]:
            document, commands = "AWS-RunPowerShellScript", windows
        else:
            document, commands = "AWS-RunShellScript", linux
        response = self.ssm[info["region"]].send_command(
            InstanceIds=[iid],
            DocumentName=document,
            Parameters={"commands": commands},
        )
        return response["Command"]["CommandId"]

    def _describe_instances(self, region: str) -> list[Any]:
        return list(self.ec2[region].get_paginator("describe_instances").paginate())

    def _describe_volumes(self, region: str) -> list[Any]:
        return list(self.ec2[region].get_paginator("describe_volumes").paginate())
