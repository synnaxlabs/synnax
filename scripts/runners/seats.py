#  Copyright 2026 Synnax Labs, Inc.
#
#  Use of this software is governed by the Business Source License included in the file
#  licenses/BSL.txt.
#
#  As of the Change Date specified in that file, in accordance with the Business Source
#  License, use of this software will be governed by the Apache License, Version 2.0,
#  included in the file licenses/APL.txt.

"""Starts the agent seats, driven by controller.py's poll cycle. A seat is one EC2
instance whose runners share an agent login. The runners belong to the organization,
so the repository's runner list holds none of them, and a seat is found by its
instance tags. A seat stops itself once idle."""

import synnax as sy

from aws import AWS
from common import RECOMMAND_DELAY
from log import Log

PROJECT = "synnax-agents"
POOL = "agent"
# Labels every seat runner carries, which name no seat.
GENERIC = {"self-hosted", "linux", "x64"}


def seat_labels(tags: dict[str, str]) -> set[str]:
    """The labels the runners of a seat register with, empty when the tags name no
    seat of the agent pool."""
    if tags.get("project") != PROJECT or tags.get("pool") != POOL:
        return set()
    return {"agent", f"agent-{tags['agent']}", f"seat-{tags['seat']}"}


class Seats:
    """A queued job starts one stopped seat that carries its labels, unless a seat
    that carries them is already up. A seat that is stopping starts once it has
    stopped. While disabled, each decision is logged once, marked [DISABLED], and no
    seat starts."""

    def __init__(self, *, aws: AWS, log: Log) -> None:
        self.aws = aws
        self.log = log
        self.started: dict[str, sy.TimeStamp] = {}
        self.held: set[str] = set()

    def step(
        self, *, queued: list[set[str]], found: dict[str, dict], enabled: bool = True
    ) -> None:
        """Starts the seats that the queued jobs, given by their labels, wait for."""
        now = sy.TimeStamp.now()
        seats = {
            iid: labels
            for iid, info in sorted(found.items())
            if (labels := seat_labels(info["tags"]))
        }
        starting: set[str] = set()
        for job in queued:
            wanted = {label.lower() for label in job} - GENERIC
            matching = [iid for iid, labels in seats.items() if wanted <= labels]
            if not wanted or any(
                iid in starting or found[iid]["state"] in ("pending", "running")
                for iid in matching
            ):
                continue
            stopped = [iid for iid in matching if found[iid]["state"] == "stopped"]
            starting.update(stopped[:1])
        if not enabled:
            for iid in sorted(starting - self.held):
                self.log(f"[DISABLED] {self.name(iid, found=found)}: start")
            self.held = starting
            return
        self.held = set()
        for iid in sorted(starting):
            last = self.started.get(iid)
            if last is not None and now.span(last) < RECOMMAND_DELAY:
                continue
            # Stamped before the call, so a failure waits out the delay.
            self.started[iid] = now
            name = self.name(iid, found=found)
            try:
                self.aws.ec2[found[iid]["region"]].start_instances(InstanceIds=[iid])
            except Exception as e:
                self.log.error(f"{name} start", e)
                continue
            self.log(f"{name}: start requested")

    def name(self, iid: str, *, found: dict[str, dict]) -> str:
        return f"seat {found[iid]['tags']['seat']}"
