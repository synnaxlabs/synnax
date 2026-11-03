#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

from typing import Any, cast

import pytest
from botocore.exceptions import ClientError
from mypy_boto3_ec2.client import EC2Client

import hook_ranges
from controller import WEBHOOK_PORT


def rule(port: int, *cidrs: str) -> dict:
    return {
        "IpProtocol": "tcp",
        "FromPort": port,
        "ToPort": port,
        "IpRanges": [{"CidrIp": c} for c in cidrs if ":" not in c],
        "Ipv6Ranges": [{"CidrIpv6": c} for c in cidrs if ":" in c],
    }


class Group:
    """Stands in for EC2 with one security group, applying the ingress rules it is
    told to add and revoke. Fails every call while failing is set."""

    def __init__(self, *rules: dict) -> None:
        self.rules = list(rules)
        self.calls: list[str] = []
        self.failing = False

    def _call(self, name: str) -> None:
        if self.failing:
            raise ClientError({"Error": {"Code": "UnauthorizedOperation"}}, name)
        self.calls.append(name)

    def describe_security_groups(self, GroupIds: list[str]) -> dict:
        self._call("describe")
        assert GroupIds == [hook_ranges.GROUP]
        return {"SecurityGroups": [{"IpPermissions": self.rules}]}

    def authorize_security_group_ingress(
        self, GroupId: str, IpPermissions: list[dict]
    ) -> None:
        self._call("authorize")
        assert GroupId == hook_ranges.GROUP
        self.rules.extend(IpPermissions)

    def revoke_security_group_ingress(
        self, GroupId: str, IpPermissions: list[dict]
    ) -> None:
        self._call("revoke")
        assert GroupId == hook_ranges.GROUP
        for p in IpPermissions:
            gone = {r["CidrIp"] for r in p["IpRanges"]} | {
                r["CidrIpv6"] for r in p["Ipv6Ranges"]
            }
            for r in self.rules:
                if r["FromPort"] != p["FromPort"]:
                    continue
                r["IpRanges"] = [x for x in r["IpRanges"] if x["CidrIp"] not in gone]
                r["Ipv6Ranges"] = [
                    x for x in r["Ipv6Ranges"] if x["CidrIpv6"] not in gone
                ]

    def open(self, port: int) -> set[str]:
        return {
            c
            for r in self.rules
            if r["FromPort"] == port
            for c in [x["CidrIp"] for x in r["IpRanges"]]
            + [x["CidrIpv6"] for x in r["Ipv6Ranges"]]
        }


def sync(group: Group, *cidrs: str) -> list[str]:
    return hook_ranges.run(ec2=cast(EC2Client, group), fetch=lambda: list(cidrs))


class TestRun:
    def test_should_add_ranges_github_lists(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "192.30.252.0/22"))
        lines = sync(group, "192.30.252.0/22", "9.234.106.48/28", "2a0a:a440::/29")
        assert group.open(WEBHOOK_PORT) == {
            "192.30.252.0/22",
            "9.234.106.48/28",
            "2a0a:a440::/29",
        }
        assert lines == ["added 2a0a:a440::/29", "added 9.234.106.48/28"]

    def test_should_describe_added_ranges(self) -> None:
        group = Group()
        sync(group, "9.234.106.48/28", "2a0a:a440::/29")
        added: dict[str, Any] = group.rules[0]
        assert added["IpRanges"][0]["Description"] == hook_ranges.DESCRIPTION
        assert added["Ipv6Ranges"][0]["Description"] == hook_ranges.DESCRIPTION

    def test_should_revoke_ranges_github_no_longer_lists(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "192.30.252.0/22", "2606:50c0::/32"))
        lines = sync(group, "192.30.252.0/22")
        assert group.open(WEBHOOK_PORT) == {"192.30.252.0/22"}
        assert lines == ["revoked 2606:50c0::/32"]

    def test_should_add_before_it_revokes(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "192.30.252.0/22"))
        sync(group, "9.234.106.48/28")
        assert group.calls == ["describe", "authorize", "revoke"]

    def test_should_change_nothing_when_in_sync(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "192.30.252.0/22"))
        assert sync(group, "192.30.252.0/22") == []
        assert group.calls == ["describe"]

    def test_should_leave_other_ports_alone(self) -> None:
        group = Group(rule(22, "203.0.113.0/24"), rule(WEBHOOK_PORT, "10.0.0.0/8"))
        sync(group, "192.30.252.0/22")
        assert group.open(22) == {"203.0.113.0/24"}
        assert group.open(WEBHOOK_PORT) == {"192.30.252.0/22"}

    def test_should_compare_normalized_ranges(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "2a0a:a440::/29"))
        assert sync(group, "2A0A:A440:0::/29") == []

    def test_should_fail_without_changes_when_github_lists_no_ranges(self) -> None:
        group = Group(rule(WEBHOOK_PORT, "192.30.252.0/22"))
        with pytest.raises(ValueError, match="no hook ranges"):
            sync(group)
        assert group.calls == []

    def test_should_raise_api_errors(self) -> None:
        group = Group()
        group.failing = True
        with pytest.raises(ClientError):
            sync(group, "192.30.252.0/22")
