#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Opens the controller's webhook port to the ranges GitHub sends webhooks from, and
closes it to ranges GitHub no longer lists.

    uv run hook_ranges.py

The Sync - Runner webhook ranges workflow runs it daily. It changes only the TCP rules
for the webhook port in the controller's security group.
"""

import ipaddress
import json
import urllib.request
from collections.abc import Callable, Iterable
from typing import TYPE_CHECKING

import boto3

from aws import TIMEOUTS
from controller import WEBHOOK_PORT
from github import SSL_CONTEXT

if TYPE_CHECKING:
    from mypy_boto3_ec2.client import EC2Client
    from mypy_boto3_ec2.type_defs import (
        IpPermissionTypeDef,
        IpRangeTypeDef,
        Ipv6RangeTypeDef,
    )

GROUP = "sg-008a955560d77bac9"
REGION = "us-east-1"
DESCRIPTION = "GitHub webhooks"


def hooks() -> list[str]:
    """The CIDRs GitHub's meta API lists under hooks."""
    req = urllib.request.Request(
        "https://api.github.com/meta",
        headers={
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        },
    )
    with urllib.request.urlopen(req, timeout=10, context=SSL_CONTEXT) as resp:
        return list(json.load(resp)["hooks"])


def _normal(cidr: str) -> str:
    return str(ipaddress.ip_network(cidr))


def _permission(cidrs: Iterable[str], *, described: bool) -> "IpPermissionTypeDef":
    v4: list[IpRangeTypeDef] = []
    v6: list[Ipv6RangeTypeDef] = []
    for c in sorted(cidrs):
        if ipaddress.ip_network(c).version == 4:
            v4.append({"CidrIp": c})
        else:
            v6.append({"CidrIpv6": c})
    if described:
        for r in [*v4, *v6]:
            r["Description"] = DESCRIPTION
    return {
        "IpProtocol": "tcp",
        "FromPort": WEBHOOK_PORT,
        "ToPort": WEBHOOK_PORT,
        "IpRanges": v4,
        "Ipv6Ranges": v6,
    }


def run(*, ec2: "EC2Client", fetch: Callable[[], list[str]]) -> list[str]:
    """Syncs the group's webhook port rules with fetch(); returns one line per change.
    Raises when fetch() returns no ranges, so an empty answer never closes the port."""
    wanted = {_normal(c) for c in fetch()}
    if not wanted:
        raise ValueError("GitHub's meta API lists no hook ranges")
    group = ec2.describe_security_groups(GroupIds=[GROUP])["SecurityGroups"][0]
    current: set[str] = set()
    for p in group["IpPermissions"]:
        if (p.get("IpProtocol"), p.get("FromPort"), p.get("ToPort")) != (
            "tcp",
            WEBHOOK_PORT,
            WEBHOOK_PORT,
        ):
            continue
        current |= {_normal(r["CidrIp"]) for r in p.get("IpRanges", [])}
        current |= {_normal(r["CidrIpv6"]) for r in p.get("Ipv6Ranges", [])}
    added, revoked = wanted - current, current - wanted
    # Adds go first, so the port is never closed to a range GitHub still uses.
    if added:
        ec2.authorize_security_group_ingress(
            GroupId=GROUP, IpPermissions=[_permission(added, described=True)]
        )
    if revoked:
        ec2.revoke_security_group_ingress(
            GroupId=GROUP, IpPermissions=[_permission(revoked, described=False)]
        )
    return [f"added {c}" for c in sorted(added)] + [
        f"revoked {c}" for c in sorted(revoked)
    ]


def main() -> None:
    ec2 = boto3.client("ec2", region_name=REGION, config=TIMEOUTS)
    for line in run(ec2=ec2, fetch=hooks) or [f"{GROUP}: in sync"]:
        print(line)


if __name__ == "__main__":
    main()
